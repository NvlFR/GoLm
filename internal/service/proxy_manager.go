package service

import (
	"math/rand"
	"strings"
	"time"

	"github.com/username/golm/internal/repository"
)

// GetRandomProxy mengambil satu proxy secara acak dari list
func GetRandomProxy() string {
	s, _ := repository.GetSettings()
	if len(s.ProxyList) == 0 {
		return ""
	}
	
	rand.Seed(time.Now().UnixNano())
	proxy := s.ProxyList[rand.Intn(len(s.ProxyList))]
	
	// Pastikan ada prefix http://
	if !strings.HasPrefix(proxy, "http") {
		return "http://" + proxy
	}
	return proxy
}

// GetProxyForAccount mengambil proxy berdasarkan index akun (Sticky per akun)
// Akun ke-1 pakai Proxy ke-1, Akun ke-2 pakai Proxy ke-2, dst.
func GetProxyForAccount(accountIndex int) string {
	s, _ := repository.GetSettings()
	if len(s.ProxyList) == 0 {
		return ""
	}
	
	// Modulo logic: Kalau akun ada 100 tapi proxy cuma 20, dia akan looping balik ke 1
	proxy := s.ProxyList[accountIndex % len(s.ProxyList)]

	if !strings.HasPrefix(proxy, "http") {
		return "http://" + proxy
	}
	return proxy
}