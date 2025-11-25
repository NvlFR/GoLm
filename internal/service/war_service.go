package service

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/username/golm/internal/antam"
	"github.com/username/golm/internal/captcha"
	"github.com/username/golm/internal/parser"
	"github.com/username/golm/internal/repository"
)

// Helper Input
func promptInput(label string, defaultValue string) string {
	fmt.Printf("%s [%s]: ", label, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func SingleWar() {
	// --- BAGIAN 1: SETUP & PILIH AKUN ---
	accounts, _ := repository.GetAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ Belum ada akun. Tambahkan dulu.")
		return
	}

	fmt.Println("\n--- 👮 PILIH AKUN ---")
	for i, acc := range accounts {
		fmt.Printf("[%d] %s\n", i+1, acc.Username)
	}
	var accIdx int
	fmt.Print("Pilih nomor: ")
	fmt.Scanln(&accIdx)
	if accIdx < 1 || accIdx > len(accounts) { return }
	
	// Pointer agar bisa diupdate saat auto-relogin
	targetAcc := &accounts[accIdx-1]
	settings, _ := repository.GetSettings()

	// --- BAGIAN 2: INPUT TARGET MANUAL ---
	// Bersihkan buffer stdin
	bufio.NewReader(os.Stdin).ReadBytes('\n') 

	fmt.Println("\n--- 🎯 KONFIGURASI TARGET ---")
	targetSiteID := promptInput("Masukkan ID Butik (3=Graha Dipta)", settings.SiteID)
	targetTimeStr := promptInput("Masukkan Jam Perang (HH:MM:SS)", settings.WarTime)

	// Init Client
	proxyURL := GetRandomProxy()
	fmt.Printf("\n🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
	client, err := antam.NewAntamClient(proxyURL)
	if err != nil {
		fmt.Println("Gagal init client:", err)
		return
	}

	// --- BAGIAN 3: SESSION CHECK & AUTO-REVIVE ---
	// Cek apakah punya cookie lama
	if len(targetAcc.Cookies) > 0 {
		client.LoadCookies(targetAcc.Cookies)
		fmt.Print("🍪 Mengecek kesehatan Cookie... ")
		
		if antam.CheckSessionAlive(client) {
			fmt.Println("✅ AKTIF!")
		} else {
			fmt.Println("❌ MATI/EXPIRED!")
			fmt.Println("🔄 Melakukan Auto-Login (Reviving)...")
			
			// Lakukan Login Ulang
			err := antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
			if err != nil {
				fmt.Printf("❌ Gagal Auto-Login: %v. Batal Perang.\n", err)
				return
			}
			// Update Cookie di Database
			saveNewSession(client, targetAcc, accIdx-1)
		}
	} else {
		fmt.Println("⚠️ Tidak ada cookie. Melakukan Login awal...")
		err := antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
		if err != nil {
			fmt.Printf("❌ Gagal Login: %v\n", err)
			return
		}
		saveNewSession(client, targetAcc, accIdx-1)
	}

	// Hitung Waktu Target
	now := time.Now()
	parsedWarTime, _ := time.Parse("15:04:05", targetTimeStr)
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), parsedWarTime.Hour(), parsedWarTime.Minute(), parsedWarTime.Second(), 0, time.Local)
	if targetTime.Before(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	fmt.Printf("\n⏳ Menunggu Waktu Perang: %s\n", targetTime.Format("15:04:05"))

	// Ambil Token Rahasia Toko (Hardcoded Map)
	secretToken, err := antam.GetTokenBySiteID(targetSiteID)
	if err != nil {
		fmt.Println("❌ Site ID tidak dikenal di database token!", targetSiteID)
		return
	}
	pageURL := fmt.Sprintf("https://antrean.logammulia.com/antrean?site=%s&t=%s", targetSiteID, secretToken)

	// --- BAGIAN 4: HEARTBEAT LOOP (JAGA LILIN) ---
	
	captchaChan := make(chan string)
	var captchaStarted bool

	for {
		timeLeft := time.Until(targetTime)
		
		// Trigger Captcha di T-90s
		if timeLeft <= 90*time.Second && !captchaStarted {
			captchaStarted = true
			go func() {
				fmt.Println("\n[Captcha] 🧩 Solving Captcha (Background)...")
				token, err := captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				if err != nil {
					fmt.Printf("❌ Gagal Captcha: %v. Retrying...\n", err)
					token, _ = captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				}
				fmt.Println("✅ Captcha SIAP!")
				captchaChan <- token
			}()
		}

		// Keluar loop di T-6s
		if timeLeft <= 6*time.Second {
			break 
		}

		// Heartbeat & Re-Check Session
		if timeLeft.Seconds() < 300 { // Mulai intensif cek saat < 5 menit
			fmt.Printf("\r[Heartbeat] 💓 Ping... (Sisa: %v) ", timeLeft.Round(time.Second))
			
			// Kita ping ke halaman antrean untuk warming up sekaligus cek login
			resp, err := client.DoRequest("GET", pageURL, nil, nil)
			var isDead bool
			if err != nil {
				fmt.Print("⚠️ Error Net ")
			} else {
				// Cek apakah dilempar ke login?
				if resp.StatusCode != 200 || strings.Contains(resp.Request.URL.String(), "login") {
					isDead = true
				}
				resp.Body.Close()
			}

			// JIKA MATI DI TENGAH JALAN -> LOGIN ULANG
			if isDead {
				fmt.Println("\n🚨 SESI MATI MENDADAK! RE-LOGIN CEPAT! 🚨")
				err := antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
				if err == nil {
					fmt.Println("✅ RE-LOGIN SUKSES! LANJUT!")
					saveNewSession(client, targetAcc, accIdx-1)
				} else {
					fmt.Printf("❌ Gagal Re-Login: %v\n", err)
				}
			}
		}

		// Sleep Logic
		sleepTime := 30 * time.Second
		if timeLeft < 60*time.Second {
			sleepTime = 5 * time.Second // Lebih sering ping saat dekat waktu
		}
		if sleepTime > timeLeft-6*time.Second {
			sleepTime = timeLeft - 6*time.Second
		}
		time.Sleep(sleepTime)
	}

	// --- BAGIAN 5: FINAL FETCH (The Critical Moment) ---
	
	fmt.Println("\n🚀 MENGAMBIL DATA SLOT TERAKHIR 🚀")
	
	// Loop Fetch Cepat (Retry 3x jika gagal extract)
	var csrfToken, finalWakdaID string
	var fetchSuccess bool

	for retry := 0; retry < 3; retry++ {
		resp, err := client.DoRequest("GET", pageURL, nil, nil)
		if err != nil {
			fmt.Println("❌ Fetch Error:", err)
			continue
		}
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(bodyBytes)

		// Save debug jika gagal nanti
		if retry == 2 {
			_ = ioutil.WriteFile("debug_final_fetch_fail.html", bodyBytes, 0644)
		}

		// 1. Parse CSRF
		cToken, err := parser.ExtractCSRF(bodyStr)
		if err != nil {
			fmt.Println("⚠️ CSRF Gagal Extract (Retrying...)")
			continue
		}
		csrfToken = cToken

		// 2. Parse Wakda (HANYA YANG AKTIF)
		// Kita cari ID wakda secara dinamis
		wakdaList, err := parser.ExtractWakda(bodyStr)
		if err != nil {
			fmt.Println("⚠️ Wakda Tidak Ditemukan (Mungkin belum buka/penuh)")
			// Jangan break, coba lagi, siapa tahu detik berikutnya muncul
			time.Sleep(500 * time.Millisecond)
			continue 
		}
		
		// Berhasil dapat Wakda List
		fmt.Printf("🔍 Ditemukan %d Slot Waktu.\n", len(wakdaList))
		for _, w := range wakdaList {
			fmt.Printf("   - [ID:%s] %s (Disabled: %v)\n", w.ID, w.Label, w.Disabled)
		}
		
		// STRATEGI: Ambil ID pertama yang ditemukan (terlepas disabled/enabled)
		// Karena saat 06:59:59 mungkin masih disabled, tapi ID itu yang akan dipakai.
		finalWakdaID = wakdaList[0].ID
		fetchSuccess = true
		break
	}

	if !fetchSuccess {
		fmt.Println("❌ GAGAL TOTAL mengambil data perang. Abort.")
		return
	}

	// Ambil Captcha
	fmt.Println("📦 Mengambil stok token captcha...")
	captchaToken := <-captchaChan

	fmt.Printf("✅ DATA LENGKAP: CSRF=%s | WAKDA=%s\n", csrfToken[:8], finalWakdaID)

	// --- BAGIAN 6: GATLING GUN (FIRE & RECORD) ---

	form := url.Values{}
	form.Set("csrf_test_name", csrfToken)
	form.Set("wakda", finalWakdaID)
	form.Set("id_cabang", targetSiteID)
	form.Set("jam_slot", targetTimeStr) // PENTING: Pakai jam yang diinput user
	form.Set("waktu", "") // Biasanya kosong
	form.Set("token", secretToken)
	form.Set("g-recaptcha-response", captchaToken)
	payload := []byte(form.Encode())

	// Burst Start
	burstStart := targetTime.Add(-200 * time.Millisecond)
	time.Sleep(time.Until(burstStart))

	fmt.Println("🔥🔥🔥 TEMBAKAN BERUNTUN DIMULAI! 🔥🔥🔥")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Duration(id*100) * time.Millisecond) // 100ms delay

			// Request
			reqStart := time.Now()
			respWar, err := client.DoRequest("POST", "https://antrean.logammulia.com/antrean-ambil", payload, map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      pageURL,
				"Origin":       "https://antrean.logammulia.com",
			})
			duration := time.Since(reqStart)

			// BLACKBOX LOGGING (Rekam Apapun Hasilnya)
			if err != nil {
				fmt.Printf("[Peluru-%d] ❌ Network Error: %v\n", id, err)
				return
			}
			defer respWar.Body.Close()
			
			bodyBytes, _ := ioutil.ReadAll(respWar.Body)
			bodyStr := string(bodyBytes)

			// Simpan Log HTML untuk analisis nanti
			logFileName := fmt.Sprintf("LOG_Peluru_%d_%d.html", id, time.Now().Unix())
			_ = ioutil.WriteFile(logFileName, bodyBytes, 0644)

			// Cek Kemenangan
			status := "❌ GAGAL"
			if strings.Contains(bodyStr, "Swal.fire") || strings.Contains(bodyStr, "qrcode") {
				status = "🏆 MENANG!!!"
			} else if strings.Contains(bodyStr, "Penuh") {
				status = "⚠️ PENUH"
			}

			fmt.Printf("[Peluru-%d] Status: %d | Time: %v | Result: %s | File: %s\n", 
				id, respWar.StatusCode, duration, status, logFileName)

		}(i)
	}
	wg.Wait()
	fmt.Println("\n🏁 WAR SELESAI. Silakan cek file HTML log.")
}

// Helper Save Session
func saveNewSession(client *antam.AntamClient, acc *repository.Account, idx int) {
	u, _ := url.Parse("https://antrean.logammulia.com")
	cookies := client.HttpClient.GetCookieJar().Cookies(u)
	
	var savedCookies []repository.CookieEntry
	for _, c := range cookies {
		savedCookies = append(savedCookies, repository.CookieEntry{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
		})
	}
	acc.Cookies = savedCookies
	repository.UpdateAccount(idx, *acc)
	fmt.Println("💾 Sesi baru disimpan ke database.")
}