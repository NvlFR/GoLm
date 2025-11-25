package service

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/username/golm/internal/antam"
	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/repository"
)

func TestProxy() {
	settings, _ := repository.GetSettings()
	
	fmt.Println("Menginisialisasi Client dengan Proxy...")
	fmt.Println("Proxy:", settings.ProxyURL)

	client, err := antam.NewAntamClient(settings.ProxyURL)
	if err != nil {
		fmt.Printf("❌ Error Init Client: %v\n", err)
		return
	}

	fmt.Println("Mencoba request ke http://ip-api.com/json (Cek IP)...")
	start := time.Now()
	
	// Kita tes ke IP checker global, bukan antam
	resp, err := client.DoRequest("GET", "http://ip-api.com/json", nil, nil)
	if err != nil {
		fmt.Printf("❌ Proxy GAGAL / Timeout: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	duration := time.Since(start)

	fmt.Printf("✅ Proxy AKTIF! (Latency: %v)\n", duration)
	fmt.Printf("Response: %s\n", string(body))
}

func TestCaptcha() {
	settings, _ := repository.GetSettings()
	
	if settings.TwoCaptchaKey == "" {
		fmt.Println("❌ API Key 2Captcha kosong di settings.json")
		return
	}

	fmt.Println("Mengirim tes captcha ke 2Captcha...")
	fmt.Println("Mohon tunggu 15-30 detik...")
	
	start := time.Now()
	token, err := captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
	
	if err != nil {
		fmt.Printf("❌ Gagal Solve: %v\n", err)
	} else {
		duration := time.Since(start)
		fmt.Printf("✅ Captcha TERPECAHKAN! (%v)\n", duration)
		fmt.Printf("Token: %s...\n", token[:20]) // Print 20 karakter awal aja
	}
}