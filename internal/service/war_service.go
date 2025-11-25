package service

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/username/golm/internal/antam"
	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
	"github.com/username/golm/internal/repository"
)

func SingleWar() {
	// --- BAGIAN 1: PERSIAPAN (Sama seperti sebelumnya) ---
	accounts, _ := repository.GetAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ Belum ada akun. Tambahkan dulu.")
		return
	}

	fmt.Println("\n--- PILIH AKUN UNTUK PERANG ---")
	for i, acc := range accounts {
		fmt.Printf("[%d] %s\n", i+1, acc.Username)
	}
	fmt.Print("Pilih nomor: ")
	var num int
	fmt.Scanln(&num)
	if num < 1 || num > len(accounts) { return }
	
	targetAcc := accounts[num-1]
	settings, _ := repository.GetSettings()

	if len(targetAcc.Cookies) == 0 {
		fmt.Println("⚠️  Akun ini belum ada cookie! Login dulu di menu [3].")
		return
	}

	proxyURL := GetRandomProxy() // Pakai proxy acak dari pool
	fmt.Printf("🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
	
	client, err := antam.NewAntamClient(proxyURL)
	if err != nil {
		fmt.Println("Gagal init client:", err)
		return
	}

	// Inject Cookie agar dianggap sudah login
	client.LoadCookies(targetAcc.Cookies)
	fmt.Println("🍪 Cookie dimuat. Siap menjaga sesi.")

	// Hitung Waktu
	now := time.Now()
	parsedWarTime, _ := time.Parse("15:04:05", settings.WarTime)
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), parsedWarTime.Hour(), parsedWarTime.Minute(), parsedWarTime.Second(), 0, time.Local)
	if targetTime.Before(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	fmt.Printf("🎯 Target Waktu: %s\n", targetTime.Format("15:04:05"))

	// Ambil Token Rahasia Toko
	secretToken, err := antam.GetTokenBySiteID(settings.SiteID)
	if err != nil {
		fmt.Println("❌ Site ID salah! Cek settings.json")
		return
	}
	pageURL := fmt.Sprintf("https://antrean.logammulia.com/antrean?site=%s&t=%s", settings.SiteID, secretToken)

	// --- BAGIAN 2: HEARTBEAT & CAPTCHA (Logic Baru) ---
	
	// Channel untuk menampung token captcha
	captchaChan := make(chan string)
	var captchaStarted bool

	fmt.Println("\n🛡️  MEMULAI FASE 'JAGA LILIN' (HEARTBEAT) 🛡️")
	fmt.Println("Bot akan me-refresh halaman setiap 30-50 detik agar tidak logout...")

	// Loop Heartbeat sampai 10 detik sebelum perang
	for {
		timeLeft := time.Until(targetTime)
		
		// A. Pemicu Captcha (Di T-90 detik)
		if timeLeft <= 90*time.Second && !captchaStarted {
			captchaStarted = true
			go func() {
				fmt.Println("\n[Captcha] 🧩 Mulai memecahkan captcha di background...")
				token, err := captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				if err != nil {
					fmt.Printf("❌ Gagal Captcha: %v\n", err)
					// Retry sekali lagi jika gagal
					token, _ = captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				}
				fmt.Println("✅ Captcha SIAP! Token disimpan.")
				captchaChan <- token
			}()
		}

		// B. Keluar Loop jika sudah dekat waktu perang (T-10 detik)
		if timeLeft <= 10*time.Second {
			break 
		}

		// C. Heartbeat Action (Refresh Halaman)
		// Logika: Akses halaman antrean untuk menjaga sesi tetap hidup
		// Sekaligus update CSRF token terbaru
		fmt.Printf("[Heartbeat] 💓 Ping server... (Sisa waktu: %v)\n", timeLeft.Round(time.Second))
		
		resp, err := client.DoRequest("GET", pageURL, nil, nil)
		if err != nil {
			fmt.Printf("⚠️ Gagal Ping: %v (Mungkin koneksi/proxy bermasalah)\n", err)
		} else {
			resp.Body.Close() // Kita cuma butuh headers cookie-nya ter-refresh
		}

		// Tidur Random 30-45 detik agar terlihat manusiawi
		sleepTime := time.Duration(30+rand.Intn(15)) * time.Second
		
		// Jangan tidur melebihi sisa waktu perang
		if sleepTime > timeLeft-10*time.Second {
			sleepTime = timeLeft - 10*time.Second
		}
		time.Sleep(sleepTime)
	}

	// --- BAGIAN 3: FINAL PREPARATION (T-5 Detik) ---
	
	fmt.Println("\n🚀 MEMASUKI FASE KRITIS (5 DETIK TERAKHIR) 🚀")
	
	// Fetch Terakhir untuk ambil CSRF & Wakda ID yang PASTI VALID
	resp, err := client.DoRequest("GET", pageURL, nil, nil)
	if err != nil {
		fmt.Println("❌ FATAL: Gagal fetch data terakhir!", err)
		return
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := string(bodyBytes)

	csrfToken, err := parser.ExtractCSRF(bodyStr)
	if err != nil {
		fmt.Println("❌ Gawat! CSRF hilang di detik terakhir.")
		return
	}

	wakdaList, err := parser.ExtractWakda(bodyStr)
	finalWakdaID := ""
	if err != nil {
		fmt.Println("⚠️ Wakda belum muncul (Toko belum buka). Menggunakan ID Prediksi.")
		// TODO: Sebaiknya ada fallback ID dari database/wakda.json
		finalWakdaID = "11" 
	} else {
		finalWakdaID = wakdaList[0].ID
	}

	// Ambil Token Captcha dari channel
	fmt.Println("[Wait] Mengambil token captcha...")
	captchaToken := <-captchaChan

	fmt.Printf("✅ DATA FINAL: CSRF=%s... | WAKDA=%s\n", csrfToken[:10], finalWakdaID)

	// --- BAGIAN 4: GATLING GUN FIRE (Strategi Rentetan) ---

	// Payload
	form := url.Values{}
	form.Set("csrf_test_name", csrfToken)
	form.Set("wakda", finalWakdaID)
	form.Set("id_cabang", settings.SiteID)
	form.Set("jam_slot", settings.WarTime)
	form.Set("waktu", "")
	form.Set("token", secretToken)
	form.Set("g-recaptcha-response", captchaToken)
	payload := []byte(form.Encode())

	// Tunggu sampai 200ms SEBELUM jam perang
	burstStartTime := targetTime.Add(-200 * time.Millisecond)
	timeToWait := time.Until(burstStartTime)
	if timeToWait > 0 {
		fmt.Printf("💣 Menahan pemicu selama %v...\n", timeToWait)
		time.Sleep(timeToWait)
	}

	fmt.Println("🔥🔥🔥 GATLING GUN: FIRE!!! 🔥🔥🔥")

	var wg sync.WaitGroup
	// Tembak 5 Peluru
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Staggering: Peluru keluar setiap 100ms
			// Peluru 1: T-200ms
			// Peluru 2: T-100ms
			// Peluru 3: T (Teng!)
			// Peluru 4: T+100ms
			// Peluru 5: T+200ms
			offset := time.Duration(id * 100) * time.Millisecond
			time.Sleep(offset)

			logPrefix := fmt.Sprintf("[Peluru-%d]", id+1)
			fmt.Printf("%s Melesat! (Offset: +%v)\n", logPrefix, offset)

			respWar, err := client.DoRequest("POST", "https://antrean.logammulia.com/antrean-ambil", payload, map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      pageURL,
				"Origin":       "https://antrean.logammulia.com",
			})

			if err != nil {
				fmt.Printf("%s ❌ Error Network: %v\n", logPrefix, err)
				return
			}
			defer respWar.Body.Close()

			// Analisis Hasil
			body, _ := ioutil.ReadAll(respWar.Body)
			sBody := string(body)

			if strings.Contains(sBody, "Swal.fire") || strings.Contains(sBody, "qrcode") {
				fmt.Printf("\n🎉🎉🎉 %s HIT TARGET! KITA MENANG! 🎉🎉🎉\n", logPrefix)
				_ = ioutil.WriteFile(fmt.Sprintf("WIN_PROOF_%d.html", id), body, 0644)
			} else if strings.Contains(sBody, "Penuh") || strings.Contains(sBody, "Habis") {
				fmt.Printf("%s ❌ Gagal: Kuota Penuh/Habis.\n", logPrefix)
			} else {
				// Simpan HTML response yang aneh untuk debug
				fmt.Printf("%s ⚠️ Status: %d (Unknown Response)\n", logPrefix, respWar.StatusCode)
				 _ = ioutil.WriteFile(fmt.Sprintf("debug_resp_%d.html", id), body, 0644)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("\n🏁 OPERASI SELESAI. Cek folder untuk bukti kemenangan.")
}