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

	// --- BAGIAN 2: PILIH BUTIK ---
	bufio.NewReader(os.Stdin).ReadBytes('\n') 

	fmt.Println("\n--- 🏢 PILIH BUTIK ---")
	siteIDs := antam.GetSiteList()
	for _, id := range siteIDs {
		name := antam.SiteNames[id]
		fmt.Printf("[%2s] %s\n", id, name)
	}
	
	fmt.Println("----------------------")
	targetSiteID := promptInput("Masukkan ID Butik", settings.SiteID)
	if _, ok := antam.SiteNames[targetSiteID]; !ok {
		fmt.Println("❌ ID Butik tidak valid!")
		return
	}
	
	targetTimeStr := promptInput("Masukkan Jam Perang (HH:MM:SS)", settings.WarTime)

	// --- BAGIAN 3: INIT CLIENT ---
	var proxyURL string
	if targetAcc.LastProxy != "" {
		proxyURL = targetAcc.LastProxy
		fmt.Printf("\n🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
		fmt.Printf("🔗 Menggunakan Proxy Terikat: ...%s\n", proxyURL[len(proxyURL)-10:])
	} else {
		proxyURL = GetRandomProxy()
		fmt.Printf("\n🔥 Menyiapkan Sniper untuk %s...\n", targetAcc.Username)
		fmt.Printf("🌍 Menggunakan Proxy Random: ...%s\n", proxyURL[len(proxyURL)-10:])
	}
	
	client, err := antam.NewAntamClient(proxyURL)
	if err != nil {
		fmt.Println("Gagal init client:", err)
		return
	}

	// Helper Re-Login
	renewSession := func() error {
		fmt.Println("🔄 ROTASI PROXY & RE-LOGIN...")
		newProxy := GetRandomProxy()
		fmt.Printf("🌍 New Proxy: ...%s\n", newProxy[len(newProxy)-10:])
		
		newClient, err := antam.NewAntamClient(newProxy)
		if err != nil { return err }
		*client = *newClient 

		err = antam.PerformLogin(client, targetAcc.Username, targetAcc.Password, settings.TwoCaptchaKey)
		if err != nil { return err }
		
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

	now := time.Now()
	parsedWarTime, _ := time.Parse("15:04:05", targetTimeStr)
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), parsedWarTime.Hour(), parsedWarTime.Minute(), parsedWarTime.Second(), 0, time.Local)
	if targetTime.Before(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	fmt.Printf("\n⏳ Target: %s @ %s\n", targetTime.Format("15:04:05"), antam.SiteNames[targetSiteID])

	secretToken, _ := antam.GetTokenBySiteID(targetSiteID)
	pageURL := fmt.Sprintf("https://antrean.logammulia.com/antrean?site=%s&t=%s", targetSiteID, secretToken)

	// --- BAGIAN 5: HEARTBEAT LOOP (DATA BACKUP) ---
	
	captchaChan := make(chan string)
	var captchaStarted bool
	var backupCSRF string // DATA CADANGAN

	for {
		timeLeft := time.Until(targetTime)
		
		if timeLeft <= 90*time.Second && !captchaStarted {
			captchaStarted = true
			go func() {
				fmt.Println("\n[Captcha] 🧩 Solving (Background)...")
				token, err := captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				if err != nil {
					fmt.Printf("❌ Gagal Captcha: %v\n", err)
					token, _ = captcha.SolveAntamCaptcha(settings.TwoCaptchaKey)
				}
				fmt.Println("✅ Captcha DONE!")
				captchaChan <- token
			}()
		}

		if timeLeft <= 6*time.Second { break }

		if timeLeft.Seconds() < 300 {
			fmt.Printf("\r[Heartbeat] 💓 Ping... (Sisa: %v) ", timeLeft.Round(time.Second))
			
			// Ping ke halaman ANTREAN untuk curi CSRF
			resp, err := client.DoRequest("GET", pageURL, nil, nil)
			var isDead bool
			
			if err != nil {
				fmt.Print("⚠️ Timeout ")
			} else {
				if resp.StatusCode != 200 || strings.Contains(resp.Request.URL.String(), "login") {
					isDead = true
				} else {
					// AMBIL CSRF BUAT CADANGAN
					bodyBytes, _ := ioutil.ReadAll(resp.Body)
					cToken, err := parser.ExtractCSRF(string(bodyBytes))
					if err == nil && cToken != "" {
						backupCSRF = cToken // Simpan untuk jaga-jaga
					}
				}
				resp.Body.Close()
			}

			if isDead {
				fmt.Println("\n🚨 SESI MATI! RE-LOGIN! 🚨")
				if err := renewSession(); err != nil {
					fmt.Printf("❌ Gagal Bangkit: %v\n", err)
				} else {
					fmt.Println("✅ BANGKIT KEMBALI!")
				}
			}
		}

		sleepTime := 30 * time.Second
		if timeLeft < 60*time.Second { sleepTime = 5 * time.Second }
		if sleepTime > timeLeft-6*time.Second { sleepTime = timeLeft - 6*time.Second }
		time.Sleep(sleepTime)
	}

	// --- BAGIAN 6: FINAL EXECUTION (RAMBO MODE) ---
	
	fmt.Println("\n🚀 FINAL FETCH DATA 🚀")
	
	var csrfToken, finalWakdaID string
	
	// Coba Fetch 3x
	for retry := 0; retry < 3; retry++ {
		resp, err := client.DoRequest("GET", pageURL, nil, nil)
		if err != nil { continue }
		
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(bodyBytes)

		if strings.Contains(bodyStr, "Log in") {
			fmt.Println("⚠️ Terlempar ke Login saat Fetch Final!")
			continue
		}

		cToken, err := parser.ExtractCSRF(bodyStr)
		if err != nil { continue }
		csrfToken = cToken

		wakdaList, err := parser.ExtractWakda(bodyStr)
		if err == nil && len(wakdaList) > 0 {
			finalWakdaID = wakdaList[0].ID
		}
		break
	}

	// LOGIC RAMBO / BLIND FIRE
	if csrfToken == "" {
		fmt.Println("⚠️ Gagal ambil CSRF baru. Menggunakan BACKUP CSRF dari Heartbeat.")
		csrfToken = backupCSRF
	}
	if finalWakdaID == "" {
		fmt.Println("⚠️ Wakda ID tidak ditemukan. Menggunakan ID PREDIKSI: 11")
		finalWakdaID = "11"
	}

	if csrfToken == "" {
		fmt.Println("❌ GAGAL TOTAL (Tidak ada CSRF). Game Over.")
		return
	}

	fmt.Println("📦 Mengambil token captcha...")
	captchaToken := <-captchaChan
	fmt.Printf("✅ DATA TEMBAK: CSRF=%s... | WAKDA=%s\n", csrfToken[:8], finalWakdaID)

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

	// Burst Time
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
			
			body, _ := ioutil.ReadAll(respWar.Body)
			sBody := string(body)
			
			// Simpan bukti
			_ = ioutil.WriteFile(fmt.Sprintf("LOG_%d.html", id), body, 0644)

			if strings.Contains(sBody, "Swal.fire") || strings.Contains(sBody, "qrcode") {
				fmt.Printf("\n🏆 P-%d MENANG! (Cek LOG_%d.html)\n", id, id)
			} else if strings.Contains(sBody, "Penuh") {
				fmt.Printf("P-%d Gagal (Penuh)\n", id)
			} else {
				fmt.Printf("P-%d Gagal (Unknown Response)\n", id)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("\n🏁 SELESAI.")
}

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