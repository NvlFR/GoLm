package antam

import (
	"bytes"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type AntamClient struct {
	HttpClient tls_client.HttpClient
	UserAgent  string
}

// NewAntamClient membuat client yang menyamar sebagai Chrome
func NewAntamClient(proxyURL string) (*AntamClient, error) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120), // Meniru Chrome v120
		tls_client.WithNotFollowRedirects(),               // Kita handle redirect manual (PENTING untuk Login)
		tls_client.WithCookieJar(jar),                     // Otomatis simpan cookies
	}

	if proxyURL != "" {
		options = append(options, tls_client.WithProxyUrl(proxyURL))
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}

	return &AntamClient{
		HttpClient: client,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}, nil
}

// DoRequest adalah wrapper untuk melakukan request HTTP
func (c *AntamClient) DoRequest(method, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	}
	if err != nil {
		return nil, err
	}

	// Standard Headers Antam (Agar terlihat natural)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Origin", "https://antrean.logammulia.com")
	req.Header.Set("Referer", "https://antrean.logammulia.com/login")
	
	// Tambahkan headers khusus jika ada
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.HttpClient.Do(req)
}