package antam

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
)

// DebugHelper: Simpan HTML ke file
func dumpHTML(filename string, body []byte) {
	_ = ioutil.WriteFile(filename, body, 0644)
	log.Printf("[Debug] HTML disimpan ke %s (Cek file ini!)", filename)
}

func PerformLogin(client *AntamClient, username, password, captchaKey string) error {
	loginURL := "https://antrean.logammulia.com/login"

	// 1. GET Login Page
	log.Println("[Auth] Mengakses halaman login...")
	resp, err := client.DoRequest("GET", loginURL, nil, nil)
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	// DEBUGGING POINT: Simpan HTML awal
	// Jika gagal extract CSRF, kamu bisa buka file ini di browser
	// untuk melihat apakah kena blokir Cloudflare.
	dumpHTML("debug_login_page.html", bodyBytes)

	// Cek apakah kena Cloudflare Turnstile
	if strings.Contains(string(bodyBytes), "Just a moment") || strings.Contains(string(bodyBytes), "turnstile") {
		return fmt.Errorf("TERDETEKSI CLOUDFLARE CHALLENGE! Proxy mungkin kotor atau TLS Client perlu ganti profil.")
	}

	// Parse CSRF
	csrfToken, err := parser.ExtractCSRF(string(bodyBytes))
	if err != nil {
		return fmt.Errorf("gagal ambil CSRF (Cek debug_login_page.html): %v", err)
	}
	log.Printf("[Auth] CSRF Login didapat: %s...", csrfToken[:10])

	// 2. Solve Captcha
	log.Println("[Auth] Solving Captcha...")
	captchaToken, err := captcha.SolveAntamCaptcha(captchaKey)
	if err != nil {
		return err
	}

	// 3. POST Login
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("token", "6137e9036fec8ee16c7dc996da46b0f16497d99f713e71315963d228f638db43") 
	formData.Set("csrf_test_name", csrfToken)
	formData.Set("g-recaptcha-response", captchaToken)
	formData.Set("rememberMe", "on")

	respLogin, err := client.DoRequest("POST", loginURL, []byte(formData.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return err
	}
	defer respLogin.Body.Close()

	if respLogin.StatusCode == 302 || respLogin.StatusCode == 303 {
		log.Println("[Auth] Login Sukses (Redirect).")
		return nil
	}

	bodyFail, _ := ioutil.ReadAll(respLogin.Body)
	dumpHTML("debug_login_fail.html", bodyFail)
	return fmt.Errorf("login gagal status %d", respLogin.StatusCode)
}