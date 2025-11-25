package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/username/golm/config"
	"github.com/username/golm/internal/antam"
	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
)

// Helper untuk menghitung selisih waktu Laptop vs Server Antam
func syncTime(client *antam.AntamClient) time.Duration {
	log.Println("[Sync] Mengkalibrasi waktu dengan server...")
	start := time.Now()
	// Gunakan HEAD request agar hemat bandwidth
	resp, err := client.DoRequest("HEAD", "https://antrean.logammulia.com/", nil, nil)
	if err != nil {
		log.Printf("[Sync] Gagal ping server: %v. Menggunakan waktu lokal.", err)
		return 0
	}
	defer resp.Body.Close()
	
	latency := time.Since(start) / 2 // Estimasi latensi satu arah

	serverDateStr := resp.Header.Get("Date")
	if serverDateStr == "" {
		log.Println("[Sync] Header Date kosong. Menggunakan waktu lokal.")
		return 0
	}

	// Parse waktu server (Format RFC1123: Mon, 02 Jan 2006 15:04:05 GMT)
	serverTime, err := time.Parse(time.RFC1123, serverDateStr)
	if err != nil {
		log.Printf("[Sync] Gagal parse waktu server: %v", err)
		return 0
	}

	// Koreksi waktu server dengan latensi
	realServerTime := serverTime.Add(latency)
	offset := realServerTime.Sub(time.Now())

	log.Printf("[Sync] Selesai. Offset: %v (Laptop kamu %v dari Server)", offset, offset)
	return offset
}

func main() {
	// 1. Load Konfigurasi
	config.LoadConfig()
	cfg := config.AppConfig

	log.Println("==============================================")
	log.Println("   🤖 GoLm - Antam Sniper Bot v1.0   ")
	log.Printf("   Target: Cabang ID %s | Pukul %s", cfg.AntamSiteID, cfg.AntamWarTime)
	log.Println("==============================================")

	// 2. Init Client (Anti-Detect)
	client, err := antam.NewAntamClient(cfg.ProxyURL)
	if err != nil {
		log.Fatal("[Init] Gagal membuat client:", err)
	}

	// 3. Kalibrasi Waktu
	timeOffset := syncTime(client)

	// Hitung Waktu Eksekusi (Adjusted)
	now := time.Now()
	parsedWarTime, _ := time.Parse("15:04:05", cfg.AntamWarTime)
	// Gabungkan Tanggal Hari Ini + Jam War
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), parsedWarTime.Hour(), parsedWarTime.Minute(), parsedWarTime.Second(), 0, time.Local)
	
	// Waktu eksekusi di laptop = Waktu Target - Offset
	// Misal: Laptop telat 2 detik. Offset = -2s.
	// Target 07:00:00. Laptop harus nembak di 06:59:58 (menurut jam laptop) agar pas di server.
	executionTime := targetTime.Add(-timeOffset)

	// Jadwal Timeline
	loginTime := targetTime.Add(-3 * time.Minute)      // 3 Menit sebelum
	captchaTime := targetTime.Add(-45 * time.Second)   // 45 Detik sebelum
	fetchDataTime := targetTime.Add(-4 * time.Second)  // 4 Detik sebelum

	log.Printf("[Jadwal] Login:   %s", loginTime.Format("15:04:05"))
	log.Printf("[Jadwal] Captcha: %s", captchaTime.Format("15:04:05"))
	log.Printf("[Jadwal] Fetch:   %s", fetchDataTime.Format("15:04:05"))
	log.Printf("[Jadwal] TEMBAK:  %s (Adjusted)", executionTime.Format("15:04:05.000"))

	// --- FASE 1: LOGIN (T-3 Menit) ---
	waitLogin := time.Until(loginTime)
	if waitLogin > 0 {
		log.Printf("[Wait] Tidur %v sampai jadwal login...", waitLogin)
		time.Sleep(waitLogin)
	}

	log.Println("[Login] Memulai proses login...")
	// Kita gunakan key captcha dari config untuk solve captcha login
	err = antam.PerformLogin(client, cfg.AntamUser, cfg.AntamPass, cfg.TwoCaptchaKey)
	if err != nil {
		log.Fatal("[Login] FATAL:", err)
	}
	log.Println("[Login] Berhasil! Sesi aman.")

	// --- FASE 2: PRE-WAR CAPTCHA (T-45 Detik) ---
	waitCaptcha := time.Until(captchaTime)
	if waitCaptcha > 0 {
		log.Printf("[Wait] Menunggu %v untuk pre-solve captcha...", waitCaptcha)
		time.Sleep(waitCaptcha)
	}

	captchaChan := make(chan string)
	go func() {
		log.Println("[Captcha] Mengirim request worker...")
		token, err := captcha.SolveAntamCaptcha(cfg.TwoCaptchaKey)
		if err != nil {
			log.Printf("[Captcha] GAGAL: %v", err)
			return
		}
		log.Printf("[Captcha] SUKSES! Token siap pakai.")
		captchaChan <- token
	}()

	// --- FASE 3: FETCH DATA TERAKHIR (T-4 Detik) ---
	// Sekaligus TCP Pre-warming
	waitFetch := time.Until(fetchDataTime)
	if waitFetch > 0 {
		time.Sleep(waitFetch)
	}

	log.Println("[Data] Mengambil data real-time (CSRF & Wakda)...")
	secretToken, _ := antam.GetTokenBySiteID(cfg.AntamSiteID)
	pageURL := fmt.Sprintf("https://antrean.logammulia.com/antrean?site=%s&t=%s", cfg.AntamSiteID, secretToken)

	resp, err := client.DoRequest("GET", pageURL, nil, nil)
	if err != nil {
		log.Fatal("[Data] Gagal fetch halaman antrean:", err)
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := string(bodyBytes)

	// Parsing
	csrfToken, err := parser.ExtractCSRF(bodyStr)
	if err != nil {
		log.Fatal("[Data] Gagal extract CSRF")
	}
	
	wakdaList, err := parser.ExtractWakda(bodyStr)
	if err != nil {
		log.Fatal("[Data] Gagal extract Wakda (Toko mungkin belum update HTML)")
	}
	
	// Ambil ID slot pertama
	finalWakdaID := wakdaList[0].ID
	log.Printf("[Data] SIAP TEMPUR! CSRF: %s... | WakdaID: %s", csrfToken[:10], finalWakdaID)

	// Tunggu Captcha Selesai
	log.Println("[Captcha] Menunggu token final...")
	finalCaptchaToken := <-captchaChan

	// --- FASE 4: GATLING GUN FIRE (T-0 Detik) ---
	
	// Siapkan Payload
	form := url.Values{}
	form.Set("csrf_test_name", csrfToken)
	form.Set("wakda", finalWakdaID)
	form.Set("id_cabang", cfg.AntamSiteID)
	form.Set("jam_slot", cfg.AntamWarTime) // Sesuai .env (07:00:00)
	form.Set("waktu", "")
	form.Set("token", secretToken)
	form.Set("g-recaptcha-response", finalCaptchaToken)
	payload := []byte(form.Encode())

	// Hitung waktu burst (Mulai 100ms sebelum target)
	burstStart := executionTime.Add(-100 * time.Millisecond)
	time.Sleep(time.Until(burstStart))

	log.Println("🔥🔥🔥 MEMULAI SERANGAN BURST! 🔥🔥🔥")

	var wg sync.WaitGroup
	// Kirim 5 peluru
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Staggering: Jeda 50ms antar peluru
			time.Sleep(time.Duration(id*50) * time.Millisecond)

			log.Printf("[Peluru-%d] Dor!", id+1)
			respWar, err := client.DoRequest("POST", "https://antrean.logammulia.com/antrean-ambil", payload, map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      pageURL,
			})
			if err != nil {
				log.Printf("[Peluru-%d] Error network: %v", id+1, err)
				return
			}
			defer respWar.Body.Close()

			resBody, _ := ioutil.ReadAll(respWar.Body)
			resStr := string(resBody)

			// ANALISIS KEMENANGAN
			if strings.Contains(resStr, "Swal.fire") || strings.Contains(resStr, "qrcode") {
				fmt.Printf("\n🎉🎉🎉 PELURU-%d HIT TARGET! ANTRIAN DIAMANKAN! 🎉🎉🎉\n", id+1)
				// Simpan bukti
				_ = ioutil.WriteFile(fmt.Sprintf("WIN_PROOF_%d.html", id), resBody, 0644)
			} else if strings.Contains(resStr, "Penuh") {
				log.Printf("[Peluru-%d] Gagal: Penuh", id+1)
			} else {
				// Debugging: simpan response aneh
				log.Printf("[Peluru-%d] Status: %d (Unknown)", id+1, respWar.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	log.Println("[Selesai] Semua peluru ditembakkan. Cek folder untuk bukti kemenangan.")
}