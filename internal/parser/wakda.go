package parser

import (
	"errors"
	"strings"

	"golang.org/x/net/html"
)

type WakdaOption struct {
	ID       string // Value (contoh: "57")
	Label    string // Text (contoh: "Pukul 08:30-11:00")
	Disabled bool   // Apakah statusnya disabled
}

// ExtractWakda mencari semua opsi di <select id="wakda">
func ExtractWakda(body string) ([]WakdaOption, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	var options []WakdaOption
	var inWakdaSelect bool

	var f func(*html.Node)
	f = func(n *html.Node) {
		// 1. Deteksi masuk ke elemen select id="wakda"
		if n.Type == html.ElementNode && n.Data == "select" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "wakda" {
					inWakdaSelect = true
					break // Ketemu parent-nya
				}
			}
		}

		// 2. Jika di dalam select wakda, cari option
		if inWakdaSelect && n.Type == html.ElementNode && n.Data == "option" {
			opt := WakdaOption{}
			for _, a := range n.Attr {
				if a.Key == "value" {
					opt.ID = a.Val
				}
				if a.Key == "disabled" {
					opt.Disabled = true
				}
			}
			// Ambil text label
			if n.FirstChild != nil {
				opt.Label = strings.TrimSpace(n.FirstChild.Data)
			}

			// Filter: Jangan ambil placeholder "--Pilih Waktu--" yang valuenya kosong
			if opt.ID != "" {
				options = append(options, opt)
			}
		}

		// Traverse anak-anak node ini
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}

		// 3. Deteksi keluar dari elemen select (backtracking)
		if n.Type == html.ElementNode && n.Data == "select" && inWakdaSelect {
			inWakdaSelect = false
		}
	}
	f(doc)

	if len(options) == 0 {
		return nil, errors.New("tidak ada opsi wakda yang ditemukan")
	}

	return options, nil
}