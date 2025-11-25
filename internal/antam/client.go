package antam

import (
	"bytes"
	"net/url"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"github.com/username/golm/internal/repository"
)

type AntamClient struct {
	HttpClient tls_client.HttpClient
	UserAgent  string
}

func NewAntamClient(proxyURL string) (*AntamClient, error) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		// FIX: Gunakan Chrome 117 (Support di tls-client v1.7.2)
		tls_client.WithClientProfile(profiles.Chrome_117),
		tls_client.WithRandomTLSExtensionOrder(),
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
		// User Agent disesuaikan ke Chrome 117
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
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

	// HEADER Chrome 117 yang valid
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="117", "Not;A=Brand";v="8", "Chromium";v="117"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", c.UserAgent)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("accept-language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	
	// Handle Referer
	if val, ok := headers["Referer"]; ok {
		req.Header.Set("referer", val)
	} else {
		req.Header.Set("referer", "https://antrean.logammulia.com/")
	}

	for k, v := range headers {
		if k != "Referer" {
			req.Header.Set(k, v)
		}
	}

	return c.HttpClient.Do(req)

	
}

func (c *AntamClient) LoadCookies(cookies []repository.CookieEntry) {
	u, _ := url.Parse("https://antrean.logammulia.com")
	var jarCookies []*http.Cookie

	for _, cEntry := range cookies {
		jarCookies = append(jarCookies, &http.Cookie{
			Name:   cEntry.Name,
			Value:  cEntry.Value,
			Domain: cEntry.Domain,
			Path:   cEntry.Path,
		})
	}
	
	c.HttpClient.GetCookieJar().SetCookies(u, jarCookies)
}