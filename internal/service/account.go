package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/username/golm/internal/repository"
)

func AddAccountMenu() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Masukkan Username (Email/NoHP): ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print("Masukkan Password: ")
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)

	newAcc := repository.Account{
		Username: user,
		Password: pass,
		Active:   true,
	}

	if err := repository.SaveAccount(newAcc); err != nil {
		fmt.Printf("❌ Gagal simpan: %v\n", err)
	} else {
		fmt.Println("✅ Akun berhasil disimpan ke database/accounts.json!")
	}
}

func DeleteAccountMenu() {
	accounts, _ := repository.GetAccounts()
	if len(accounts) == 0 {
		fmt.Println("Belum ada akun tersimpan.")
		return
	}

	fmt.Println("Daftar Akun:")
	for i, acc := range accounts {
		fmt.Printf("[%d] %s\n", i+1, acc.Username)
	}

	fmt.Print("\nPilih nomor akun untuk dihapus (0 untuk batal): ")
	var num int
	fmt.Scanln(&num)

	if num > 0 && num <= len(accounts) {
		err := repository.DeleteAccount(num - 1)
		if err != nil {
			fmt.Println("❌ Gagal hapus:", err)
		} else {
			fmt.Println("✅ Akun dihapus.")
		}
	}
}

func ViewSettings() {
	s, _ := repository.GetSettings()
	fmt.Println("\n--- KONFIGURASI SAAT INI ---")
	// PERBAIKAN DI SINI:
	// Kita hitung jumlah proxy di list, bukan print string URL
	fmt.Printf("Proxy Pool : %d IPs Loaded\n", len(s.ProxyList)) 
	fmt.Printf("Site ID    : %s\n", s.SiteID)
	fmt.Printf("War Time   : %s\n", s.WarTime)
	fmt.Printf("2Captcha   : %s\n", s.TwoCaptchaKey)
	fmt.Println("\n(Edit file database/settings.json untuk mengubah)")
}