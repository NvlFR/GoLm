package parser

import (
	"errors"
	"strings"

	"golang.org/x/net/html"
)

// ExtractCSRF mencari input hidden dengan name="csrf_test_name"
// Ini jauh lebih cepat dan akurat daripada Regex
func ExtractCSRF(body string) (string, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", err
	}

	var csrfToken string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				}
				if a.Key == "value" {
					value = a.Val
				}
			}
			if name == "csrf_test_name" {
				csrfToken = value
			}
		}
		// Recursive traverse
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if csrfToken == "" {
		return "", errors.New("csrf token not found in html")
	}

	return csrfToken, nil
}