package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/username/golm/config"
	"github.com/username/golm/internal/repository"
	"github.com/username/golm/internal/service"
)

func main() {
	// 1. Init System
	config.LoadConfig()
	repository.InitDB()
	
	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreen()
		// Baca settings terbaru setiap kali refresh menu
		currentSettings, _ := repository.GetSettings()
		
		printBanner(currentSettings)
		printMenu()

		fmt.Print("\n[?] Masukkan Pilihan (0-11): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		handleInput(input)

		fmt.Println("\nTekan [Enter] untuk kembali ke menu...")
		reader.ReadString('\n')
	}
}

func printBanner(s repository.Settings) {
// Cek Status Proxy List
	proxyStatus := "❌ OFF"
	proxyCount := len(s.ProxyList)
	if proxyCount > 0 {
		proxyStatus = fmt.Sprintf("✅ ON (%d IPs Loaded)", proxyCount)
	}

	// Cek Status Captcha
	captchaStatus := "❌ OFF"
	if len(s.TwoCaptchaKey) > 5 {
		captchaStatus = "✅ ON"
	}
	
	// Cek Mode Debug
	debugStatus := "MATI"
	if s.Debug {
		debugStatus = "NYALA "
	}

	fmt.Println("==================================================")
	fmt.Println("          🤖 GOLM - COMMAND CENTER v1.2          ")
	fmt.Println("==================================================")
	// Tampilan Dashboard Status
	fmt.Printf(" [📡 Proxy]    : %s\n", proxyStatus)
	fmt.Printf(" [🧩 2Captcha] : %s\n", captchaStatus)
	fmt.Printf(" [🐞 Debug]    : %s\n", debugStatus)
	fmt.Println("--------------------------------------------------")
	fmt.Printf(" [🎯 Target]   : Cabang ID %s\n", s.SiteID)
	fmt.Printf(" [⏰ Waktu War]: %s\n", s.WarTime)
	fmt.Println("==================================================")
}

func printMenu() {
	fmt.Println("[1]  🚀 Perang Single Akun")
	fmt.Println("[2]  🚀 Perang Multi Akun")
	fmt.Println("------------------------------------------")
	fmt.Println("[3]  🔑 Login Single Akun")
	fmt.Println("[4]  🔑 Login Semua Akun")
	fmt.Println("------------------------------------------")
	fmt.Println("[5]  📊 Cek Slot & Kuota")
	fmt.Println("------------------------------------------")
	fmt.Println("[6]  👤 Tambah Akun")
	fmt.Println("[7]  🗑️ Hapus Akun")
	fmt.Println("[8]  ⚙️  Lihat Setting Lengkap")
	fmt.Println("[9]  🕵️ Scrape Wakda ID")
	fmt.Println("------------------------------------------")
	fmt.Println("[10] 📡 Tes Proxy (Cek IP)")
	fmt.Println("[11] 🧩 Tes 2Captcha")
	fmt.Println("[0]  ❌ Keluar")
}

func handleInput(choice string) {
	switch choice {
	case "1":
		// service.SingleWar() // Nanti kita buka comment ini
		fmt.Println("Fitur Perang Single (Menunggu Server Buka besok pagi)")
	case "2":
		fmt.Println("Fitur Perang Multi (Menunggu Server Buka besok pagi)")
	case "3":
		service.LoginSingleAccount()
	case "4":
		fmt.Println("Fitur Login Multi sedang dikerjakan...")
	case "5":
		fmt.Println("Fitur Cek Slot sedang dikerjakan...")
	case "6":
		service.AddAccountMenu()
	case "7":
		service.DeleteAccountMenu()
	case "8":
		service.ViewSettings()
	case "9":
		fmt.Println("Fitur Scrape Wakda (Menunggu Server Buka besok pagi)")
	case "10":
		service.TestProxy()
	case "11":
		service.TestCaptcha()
	case "0":
		fmt.Println("Bye bye, Engineer!")
		os.Exit(0)
	default:
		fmt.Println("⚠️ Pilihan tidak valid!")
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}