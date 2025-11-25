package antam

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
)

// PerformLogin melakukan flow login lengkap
func PerformLogin(client *AntamClient, username, password, captchaKey string) error {
	loginURL := "https://antrean.logammulia.com/login"

	// 1. GET Login Page (Untuk ambil CSRF Token awal & Cookies Session)
	// Cloudflare challenge biasanya terjadi di sini. tls-client akan menanganinya.
	resp, err := client.DoRequest("GET", loginURL, nil, nil)
	if err != nil {
		return fmt.Errorf("gagal akses halaman login: %v", err)
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Parse CSRF dari HTML
	csrfToken, err := parser.ExtractCSRF(string(bodyBytes))
	if err != nil {
		return fmt.Errorf("gagal ambil CSRF login: %v", err)
	}
	log.Printf("[Auth] CSRF Login didapat: %s...", csrfToken[:10])

	// 2. Solve Captcha (Paralel dengan persiapan data)
	log.Println("[Auth] Memecahkan Captcha Login...")
	captchaToken, err := captcha.SolveAntamCaptcha(captchaKey)
	if err != nil {
		return fmt.Errorf("gagal solve captcha: %v", err)
	}
	log.Println("[Auth] Captcha Terpecahkan.")

	// 3. POST Login Data
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("token", "6137e9036fec8ee16c7dc996da46b0f16497d99f713e71315963d228f638db43") // Token statis dari log kamu
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

	// 4. Validasi Login
	// Antam me-redirect (302/303) ke /users jika sukses
	if respLogin.StatusCode == 302 || respLogin.StatusCode == 303 {
		location, _ := respLogin.Header["Location"]
		log.Printf("[Auth] Login Sukses! Redirect ke: %v", location)
		return nil
	}

	// Jika 200 OK berarti masih di halaman login (Gagal)
	bodyLogin, _ := ioutil.ReadAll(respLogin.Body)
	if len(bodyLogin) < 500 {
		// Log error pendek
		return fmt.Errorf("login gagal, status: %d response: %s", respLogin.StatusCode, string(bodyLogin))
	}
	
	return fmt.Errorf("login gagal, status: %d (kemungkinan salah pass/captcha)", respLogin.StatusCode)
}