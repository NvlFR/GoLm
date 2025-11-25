package captcha

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Antam Constants
const (
	SiteKey = "6LdxTgUsAAAAAJ80-chHLt5PiK-xv1HbLPqQ3nB4" // Sitekey Login & War sama
	PageURL = "https://antrean.logammulia.com/login"      // URL target
)

// SolveAntamCaptcha mengirim request ke 2captcha dan menunggu hasil
func SolveAntamCaptcha(apiKey string) (string, error) {
	if apiKey == "" {
		return "", errors.New("API Key 2Captcha kosong")
	}

	// 1. Kirim Task ke 2Captcha
	reqURL := fmt.Sprintf("http://2captcha.com/in.php?key=%s&method=userrecaptcha&googlekey=%s&pageurl=%s", apiKey, SiteKey, PageURL)
	resp, err := http.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	respStr := string(body)

	if !strings.HasPrefix(respStr, "OK|") {
		return "", fmt.Errorf("gagal mengirim captcha: %s", respStr)
	}

	requestID := strings.Split(respStr, "|")[1]
	// fmt.Println("Captcha ID:", requestID) // Debug only

	// 2. Polling Hasil (Maksimal 120 detik)
	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)

		pollURL := fmt.Sprintf("http://2captcha.com/res.php?key=%s&action=get&id=%s", apiKey, requestID)
		pollResp, err := http.Get(pollURL)
		if err != nil {
			continue
		}
		
		pollBody, _ := ioutil.ReadAll(pollResp.Body)
		pollStr := string(pollBody)
		pollResp.Body.Close()

		if pollStr == "CAPCHA_NOT_READY" {
			continue
		}

		if strings.HasPrefix(pollStr, "OK|") {
			return strings.Split(pollStr, "|")[1], nil // TOKEN BERHASIL
		}
		
		if strings.HasPrefix(pollStr, "ERROR") {
			return "", fmt.Errorf("2captcha error: %s", pollStr)
		}
	}

	return "", errors.New("timeout menunggu captcha")
}