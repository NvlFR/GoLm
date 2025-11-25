package service

import (
	"fmt"
	"net/url"

	"github.com/username/golm/internal/antam"
	"github.com/username/golm/internal/repository"
)

func LoginSingleAccount() {
	// Ambil semua akun
	accounts, _ := repository.GetAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ Tidak ada akun! Tambahkan akun dulu di menu [6].")
		return
	}

	// Tampilkan pilihan
	fmt.Println("Pilih akun untuk login:")
	for i, acc := range accounts {
		fmt.Printf("[%d] %s\n", i+1, acc.Username)
	}
	
	fmt.Print("Pilih nomor: ")
	var num int
	fmt.Scanln(&num)

	if num < 1 || num > len(accounts) {
		fmt.Println("Pilihan salah.")
		return
	}

	targetAcc := accounts[num-1]
	settings, _ := repository.GetSettings()

	proxyURL := GetRandomProxy()
	fmt.Printf("Mencoba login %s menggunakan Proxy: ...%s\n", targetAcc.Username, proxyURL[len(proxyURL)-10:]) // Tampilkan buntutnya aja
	// Init Client
	client, err := antam.NewAntamClient(proxyURL)
	if err != nil {
		fmt.Println("Error Client:", err)
		return
	}

	err = antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
	if err != nil {
		fmt.Printf("❌ GAGAL LOGIN: %v\n", err)
	} else {
		fmt.Println("✅ BERHASIL LOGIN! Menyimpan sesi ke database...")

		// --- LOGIC PENYIMPANAN COOKIE ---
		targetURL, _ := url.Parse("https://antrean.logammulia.com")
		// Ambil cookie dari Jar si Client
		cookies := client.HttpClient.GetCookieJar().Cookies(targetURL)
		
		// Konversi ke format database kita
		var savedCookies []repository.CookieEntry
		for _, c := range cookies {
			// FIX: Pakai domain yang lebih umum agar cookie terbaca di semua subdomain
			domain := c.Domain
			if domain == "" || domain == "antrean.logammulia.com" {
				domain = ".logammulia.com" 
			}

			savedCookies = append(savedCookies, repository.CookieEntry{
				Name:   c.Name,
				Value:  c.Value,
				Domain: domain, 
				Path:   "/",
			})
		}

		// Update struct akun
		targetAcc.Cookies = savedCookies
		targetAcc.LastProxy = proxyURL
		
		// Simpan ke JSON
		err := repository.UpdateAccount(num-1, targetAcc)
		if err != nil {
			fmt.Printf("⚠️ Login sukses tapi gagal simpan ke file: %v\n", err)
		} else {
			fmt.Printf("💾 Sesi disimpan! (%d cookies aktif)\n", len(savedCookies))
		}
	}
}