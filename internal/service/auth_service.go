package service

import (
	"fmt"

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

	// Panggil Logic Login Low-Level
	err = antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
	if err != nil {
		fmt.Printf("❌ GAGAL LOGIN: %v\n", err)
	} else {
		fmt.Println("✅ BERHASIL LOGIN! Cookies tersimpan di memory.")
	}
}