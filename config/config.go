package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Struct untuk menampung semua config
type Config struct {
	AntamUser      string
	AntamPass      string
	AntamSiteID    string
	AntamWarTime   string
	ProxyURL       string
	TwoCaptchaKey  string
}

// Global variable yang bisa diakses package lain
var AppConfig *Config

func LoadConfig() {
	// Load file .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Tidak menemukan file .env, pastikan environment variable sudah diset.")
	}

	AppConfig = &Config{
		AntamUser:      os.Getenv("ANTAM_USERNAME"),
		AntamPass:      os.Getenv("ANTAM_PASSWORD"),
		AntamSiteID:    os.Getenv("ANTAM_SITE_ID"),
		AntamWarTime:   os.Getenv("ANTAM_WAR_TIME"),
		ProxyURL:       os.Getenv("PROXY_URL"),
		TwoCaptchaKey:  os.Getenv("TWOCAPTCHA_KEY"),
	}

	// Validasi sederhana
	if AppConfig.AntamUser == "" || AppConfig.ProxyURL == "" {
		log.Fatal("Error: Konfigurasi .env belum lengkap! Username atau Proxy kosong.")
	}
}