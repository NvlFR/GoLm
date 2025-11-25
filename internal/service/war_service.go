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
	// --- BAGIAN 1: SETUP AKUN ---
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
	
	targetAcc := &accounts[accIdx-1]
	settings, _ := repository.GetSettings()

	// --- BAGIAN 2: PILIH BUTIK (FITUR BARU: TAMPILKAN LIST) ---
	// Bersihkan buffer
	bufio.NewReader(os.Stdin).ReadBytes('\n') 

	fmt.Println("\n--- 🏢 PILIH BUTIK ---")
	// Menampilkan list butik dari tokens.go
	siteIDs := antam.GetSiteList()
	for _, id := range siteIDs {
		name := antam.SiteNames[id]
		fmt.Printf("[%2s] %s\n", id, name)
	}
	
	fmt.Println("----------------------")
	targetSiteID := promptInput("Masukkan ID Butik", settings.SiteID)
	
	// Validasi ID
	if _, ok := antam.SiteNames[targetSiteID]; !ok {
		fmt.Println("❌ ID Butik tidak valid! Pastikan memilih angka yang ada di list.")
		return
	}
	
	targetTimeStr := promptInput("Masukkan Jam Perang (HH:MM:SS)", settings.WarTime)

// --- BAGIAN 3: INIT CLIENT (STRICT IP BINDING) ---
	
	var proxyURL string
	
	// Cek apakah akun punya riwayat proxy?
	if targetAcc.LastProxy != "" {
		proxyURL = targetAcc.LastProxy
		fmt.Printf("\n🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
		fmt.Printf("🔗 Menggunakan Proxy Terikat (Sesi Lama): ...%s\n", proxyURL[len(proxyURL)-10:])
	} else {
		// Kalau belum pernah login/gak ada data, baru ambil random
		proxyURL = GetRandomProxy()
		fmt.Printf("\n🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
		fmt.Printf("🌍 Menggunakan Proxy Random: ...%s\n", proxyURL[len(proxyURL)-10:])
	}
	
	client, err := antam.NewAntamClient(proxyURL)
	if err != nil {
		fmt.Println("Gagal init client:", err)
		return
	}

	// --- BAGIAN 4: SESSION CHECK & AUTO-REVIVE (DENGAN ROTASI PROXY) ---
	
	// Helper Function untuk Re-Login Total dengan Proxy Baru
	renewSession := func() error {
		fmt.Println("🔄 ROTASI PROXY & RE-LOGIN...")
		
		// 1. Ganti Proxy (Karena yang lama mungkin sudah diblokir)
		newProxy := GetRandomProxy()
		fmt.Printf("🌍 New Proxy: ...%s\n", newProxy[len(newProxy)-10:])
		
		// 2. Buat Client Baru
		newClient, err := antam.NewAntamClient(newProxy)
		if err != nil { return err }
		*client = *newClient 

		// 3. Login Ulang
		err = antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
		if err != nil { return err }
		
		// 4. Simpan Sesi DAN Proxy Baru ke Database
		// Update LastProxy di struct lokal dulu
		targetAcc.LastProxy = newProxy
		saveNewSession(client, targetAcc, accIdx-1)
		return nil
	}

	if len(targetAcc.Cookies) > 0 {
		client.LoadCookies(targetAcc.Cookies)
		fmt.Print("🍪 Cek Sesi... ")
		if antam.CheckSessionAlive(client) {
			fmt.Println("✅ AKTIF!")
		} else {
			fmt.Println("❌ MATI!")
			if err := renewSession(); err != nil {
				fmt.Printf("❌ Gagal Revive: %v. Abort.\n", err)
				return
			}
		}
	} else {
		fmt.Println("⚠️ Sesi kosong.")
		if err := renewSession(); err != nil {
			fmt.Printf("❌ Gagal Login Awal: %v\n", err)
			return
		}
	}

	// Hitung Waktu
	now := time.Now()
	parsedWarTime, _ := time.Parse("15:04:05", targetTimeStr)
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), parsedWarTime.Hour(), parsedWarTime.Minute(), parsedWarTime.Second(), 0, time.Local)
	if targetTime.Before(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	fmt.Printf("\n⏳ Target: %s @ %s\n", targetTime.Format("15:04:05"), antam.SiteNames[targetSiteID])

	secretToken, _ := antam.GetTokenBySiteID(targetSiteID)
	pageURL := fmt.Sprintf("https://antrean.logammulia.com/antrean?site=%s&t=%s", targetSiteID, secretToken)

	// --- BAGIAN 5: HEARTBEAT LOOP (JAGA LILIN) ---
	
	captchaChan := make(chan string)
	var captchaStarted bool

	for {
		timeLeft := time.Until(targetTime)
		
		// Trigger Captcha (T-90 detik)
		if timeLeft <= 90*time.Second && !captchaStarted {
			captchaStarted = true
			go func() {
				fmt.Println("\n[Captcha] 🧩 Solving (Background)...")
				token, err := captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				if err != nil {
					fmt.Printf("❌ Gagal Captcha: %v. Retrying...\n", err)
					token, _ = captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				}
				fmt.Println("✅ Captcha SIAP!")
				captchaChan <- token
			}()
		}

		if timeLeft <= 6*time.Second { break }

		// Heartbeat dengan Auto-Heal
		if timeLeft.Seconds() < 300 {
			fmt.Printf("\r[Heartbeat] 💓 Ping... (Sisa: %v) ", timeLeft.Round(time.Second))
			
			// UBAH: Ping ke halaman Users (lebih aman buat cek sesi)
			pingURL := "https://antrean.logammulia.com/users"
			resp, err := client.DoRequest("GET", pingURL, nil, nil)
			
			var isDead bool
			
			if err != nil {
				// Kalau timeout/error network, JANGAN panik. Anggap masih hidup.
				fmt.Print("⚠️ Lag (Ignored) ")
			} else {
				// Hanya mati jika status 302/303 DAN Location mengandung 'login'
				// ATAU jika body HTML mengandung form login
				if resp.StatusCode == 302 || resp.StatusCode == 303 {
					loc, _ := resp.Header["Location"]
					if len(loc) > 0 && strings.Contains(loc[0], "login") {
						isDead = true
					}
				} else if resp.StatusCode == 200 {
					// Baca sedikit body untuk memastikan tidak ada form login
					// (Opsional, tapi bagus untuk akurasi)
					bodyPreview, _ := ioutil.ReadAll(resp.Body)
					if strings.Contains(string(bodyPreview), "Masukan e-mail") {
						isDead = true
					}
				}
				resp.Body.Close()
			}

			if isDead {
				fmt.Println("\n🚨 SESI BENAR-BENAR MATI (LOGOUT). RE-LOGIN! 🚨")
				if err := renewSession(); err != nil {
					fmt.Printf("❌ Gagal Bangkit: %v (Retrying next loop)\n", err)
				} else {
					fmt.Println("✅ BANGKIT KEMBALI!")
				}
			}
		}
        // ... (kode sleep sama) ...

		// Jeda Heartbeat
		sleepTime := 30 * time.Second
		if timeLeft < 60*time.Second { sleepTime = 5 * time.Second }
		if sleepTime > timeLeft-6*time.Second { sleepTime = timeLeft - 6*time.Second }
		time.Sleep(sleepTime)
	}

	// --- BAGIAN 6: FINAL EXECUTION (GATLING GUN) ---
	
	fmt.Println("\n🚀 FINAL FETCH DATA 🚀")
	
	var csrfToken, finalWakdaID string
	var fetchSuccess bool

	// Retry logic agresif untuk ambil data terakhir
	for retry := 0; retry < 3; retry++ {
		resp, err := client.DoRequest("GET", pageURL, nil, nil)
		if err != nil { continue }
		
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(bodyBytes)

		// Check apakah malah dilempar ke login lagi?
		if strings.Contains(bodyStr, "Log in") || strings.Contains(bodyStr, "turnstile") {
			fmt.Println("⚠️ Terlempar ke Login saat Fetch Final! (Panic Mode)")
			continue
		}

		cToken, err := parser.ExtractCSRF(bodyStr)
		if err != nil { continue }
		csrfToken = cToken

		wakdaList, err := parser.ExtractWakda(bodyStr)
		if err != nil {
			// Fallback ID jika toko belum buka di HTML (tapi API mungkin sudah siap)
			finalWakdaID = "11" 
		} else {
			finalWakdaID = wakdaList[0].ID
		}
		
		fetchSuccess = true
		break
	}

	if !fetchSuccess {
		fmt.Println("❌ GAGAL FETCH FINAL. Mencoba Blind Fire...")
		return
	}

	fmt.Println("📦 Mengambil stok token captcha...")
	captchaToken := <-captchaChan
	fmt.Printf("✅ DATA LENGKAP: CSRF=%s | WAKDA=%s\n", csrfToken[:8], finalWakdaID)

	// Payload
	form := url.Values{}
	form.Set("csrf_test_name", csrfToken)
	form.Set("wakda", finalWakdaID)
	form.Set("id_cabang", targetSiteID)
	form.Set("jam_slot", targetTimeStr)
	form.Set("waktu", "")
	form.Set("token", secretToken)
	form.Set("g-recaptcha-response", captchaToken)
	payload := []byte(form.Encode())

	// Burst Time (Mulai 200ms sebelum target)
	burstStart := targetTime.Add(-200 * time.Millisecond)
	time.Sleep(time.Until(burstStart))

	fmt.Println("🔥🔥🔥 FIRE (GATLING MODE) 🔥🔥🔥")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Duration(id*100) * time.Millisecond)

			respWar, err := client.DoRequest("POST", "https://antrean.logammulia.com/antrean-ambil", payload, map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      pageURL,
				"Origin":       "https://antrean.logammulia.com",
			})

			if err != nil {
				fmt.Printf("P-%d ❌ Err\n", id)
				return
			}
			defer respWar.Body.Close()
			
			// SIMPAN BUKTI HTML
			body, _ := ioutil.ReadAll(respWar.Body)
			sBody := string(body)
			_ = ioutil.WriteFile(fmt.Sprintf("LOG_%d.html", id), body, 0644)

			if strings.Contains(sBody, "Swal.fire") || strings.Contains(sBody, "qrcode") {
				fmt.Printf("\n🏆 P-%d MENANG! (Cek LOG_%d.html)\n", id, id)
			} else {
				fmt.Printf("P-%d Gagal\n", id)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("\n🏁 WAR SELESAI.")
}

// Helper untuk menyimpan sesi baru ke database setelah auto-login
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