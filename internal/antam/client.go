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

func NewAntamClient(proxyURL string) (*AntamClient, error) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		// UBAH 1: Ganti Profil ke Safari MacOS (Lebih trusted)
		tls_client.WithClientProfile(profiles.Safari_15_6_1), 
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
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
		// UBAH 2: User Agent harus PERSIS sama dengan profil Safari di atas
		UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15",
	}, nil
}

func (c *AntamClient) DoRequest(method, url string, body []byte, headers map[string]string) (*http.Response, error) {
	var req *http.Request
	var err error

	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	// UBAH 3: Headers harus meniru Safari
	// Urutan header itu PENTING bagi Cloudflare
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br") // Biarkan library yang atur ini otomatis
	
	// Header tambahan
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// Timpa dengan header khusus (seperti Referer dll)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.HttpClient.Do(req)
}