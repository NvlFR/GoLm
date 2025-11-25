package antam

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
)

// DebugHelper
func dumpHTML(filename string, body []byte) {
	_ = ioutil.WriteFile(filename, body, 0644)
}


func PerformLogin(client *AntamClient, username, password, captchaKey string) error {
	loginURL := "https://antrean.logammulia.com/login"
	homeURL := "https://antrean.logammulia.com/"

	// ---------------------------------------------------------
	// LANGKAH 0: WARMING UP (PENTING UNTUK BYPASS CLOUDFLARE)
	// ---------------------------------------------------------
	log.Println("[Auth] Mengakses Homepage untuk inisialisasi Cookie...")
	respHome, err := client.DoRequest("GET", homeURL, nil, nil)
	if err == nil {
		respHome.Body.Close()
		// Sleep sebentar seolah-olah loading page
		time.Sleep(2 * time.Second)
	}
	
	// ---------------------------------------------------------
	// LANGKAH 1: GET LOGIN PAGE
	// ---------------------------------------------------------
	log.Println("[Auth] Mengakses halaman login...")
	resp, err := client.DoRequest("GET", loginURL, nil, nil)
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Cek Jebakan Cloudflare
	bodyString := string(bodyBytes)
	if strings.Contains(bodyString, "Just a moment") || strings.Contains(bodyString, "turnstile") {
		dumpHTML("debug_cf_blocked.html", bodyBytes)
		return fmt.Errorf("TERBLOKIR: Cloudflare Challenge muncul. Coba ganti Proxy.")
	}

	// Parse CSRF
	csrfToken, err := parser.ExtractCSRF(bodyString)
	if err != nil {
		dumpHTML("debug_no_csrf.html", bodyBytes)
		return fmt.Errorf("gagal ambil CSRF: %v", err)
	}
	log.Printf("[Auth] CSRF Login didapat: %s...", csrfToken[:10])

	// ---------------------------------------------------------
	// LANGKAH 2: SOLVE CAPTCHA (Paralel)
	// ---------------------------------------------------------
	log.Println("[Auth] Solving Captcha...")
	captchaToken, err := captcha.SolveAntamCaptcha(captchaKey)
	if err != nil {
		return err
	}
	// Sleep random biar kayak manusia yang baru selesai ngetik captcha
	time.Sleep(1 * time.Second)

	// ---------------------------------------------------------
	// LANGKAH 3: POST LOGIN
	// ---------------------------------------------------------
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("token", "6137e9036fec8ee16c7dc996da46b0f16497d99f713e71315963d228f638db43") 
	formData.Set("csrf_test_name", csrfToken)
	formData.Set("g-recaptcha-response", captchaToken)
	formData.Set("rememberMe", "on")

	// Header Referer PENTING saat POST
	headers := map[string]string{
		"Referer": loginURL,
		"Origin": "https://antrean.logammulia.com",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	respLogin, err := client.DoRequest("POST", loginURL, []byte(formData.Encode()), headers)
	if err != nil {
		return err
	}
	defer respLogin.Body.Close()

	// ---------------------------------------------------------
	// LANGKAH 4: VALIDASI
	// ---------------------------------------------------------
	// Sukses biasanya redirect 302/303
	if respLogin.StatusCode == 302 || respLogin.StatusCode == 303 {
		log.Println("[Auth] Login Sukses (Redirect). Sesi disimpan.")
		return nil
	}

	// Cek response body jika tidak redirect
	bodyFail, _ := ioutil.ReadAll(respLogin.Body)
	if strings.Contains(string(bodyFail), "dashboard") || strings.Contains(string(bodyFail), "users") {
		log.Println("[Auth] Login Sukses (200 OK).")
		return nil
	}

	dumpHTML("debug_login_fail.html", bodyFail)
	return fmt.Errorf("login gagal status %d", respLogin.StatusCode)
}

func CheckSessionAlive(client *AntamClient) bool {
	// Kita cek ke halaman antrean, bukan users (lebih ringan)
	// Dan pakai header lengkap agar tidak dikira bot
	resp, err := client.DoRequest("GET", "https://antrean.logammulia.com/antrean", nil, nil)
	if err != nil {
		// Network error jangan dianggap sesi mati dulu (bisa jadi proxy timeout)
		return false 
	}
	defer resp.Body.Close()

	// Jika status 200 OK, kemungkinan besar masih hidup
	if resp.StatusCode == 200 {
		// Cek redirect URL (Kalau dilempar ke login, berarti mati)
		finalURL := resp.Request.URL.String()
		if strings.Contains(finalURL, "login") {
			return false
		}
		return true
	}

	// Jika 302/303 redirect ke login -> Mati
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		loc, _ := resp.Header["Location"]
		if len(loc) > 0 && strings.Contains(loc[0], "login") {
			return false
		}
	}

	// Default: Asumsikan hidup kalau responnya aneh-aneh (biar gak dikit-dikit login)
	return true
}
