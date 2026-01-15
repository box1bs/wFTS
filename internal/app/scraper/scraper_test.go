package scraper

import (
	"context"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/box1bs/wFTS/pkg/logger"
)

func TestHtmlGetter(t *testing.T) {
	ws := NewScraper(nil, &ConfigData{}, logger.NewLogger(nil, os.Stdout, 1), nil, nil, context.Background(), nil)
	html, err := ws.getHTML("https://example.com/", NewRateLimiter(1), 3)
	if err != nil {
		t.Errorf("error while gitting html: %v", err)
		return
	}
	htmlReader, err := os.OpenFile("/home/box1bs/.projects/monocle/internal/assets/utest.html", os.O_RDONLY, 0600)
	if err != nil {
		t.Errorf("error while reading html file by path `/home/box1bs/.projects/monocle/internal/assets/utest.html` with error: %v", err)
		return
	}
	bytes, err := io.ReadAll(htmlReader)
	if err != nil {
		t.Errorf("error reading bytes from file: %v", err)
		return
	}
	if string(bytes) != html {
		t.Errorf("unexpected html: %s, instead if %s", html, bytes)
	}
}

func TestHaveSitemap(t *testing.T) {
	parsed, err := url.Parse("https://www.google.com/")
	if err != nil {
		t.Errorf("error parsing url: %v", err)
		return
	}
	links, err := NewScraper(nil, &ConfigData{}, nil, nil, nil, context.Background(), nil).haveSitemap(parsed)
	if err != nil {
		t.Errorf("error handling sitemap: %v", err)
		return
	}
	if l := len(links); l != 21 {
		t.Errorf("not enough links in sitemap: %d", l)
	}
}

func TestNormalizeUrl(t *testing.T) {
	wwwTest := "https://www.example.com/"
	slashTest := "https://example.com//"
	queryTest := "https://example.com/?id=1"
	if t1Value, err := normalizeUrl(wwwTest); t1Value != "example.com" || err != nil {
		t.Errorf("invalid normalization: %s, or error: %v", t1Value, err)
		return
	}
	if t2Value, err := normalizeUrl(slashTest); t2Value != "example.com" || err != nil {
		t.Errorf("invalid normalization: %s, or error: %v", t2Value, err)
		return
	}
	if t3Value, err := normalizeUrl(queryTest); t3Value != "example.com?id=1" || err != nil {
		t.Errorf("invalid normalization: %s, or error: %v", t3Value, err)
	}
}

func TestDomainDefinition(t *testing.T) {
	url1 := "https://www.google.com/"
	url2 := "https://www.google.com/search?q=thfjngjkyk&sca_esv=492d03d456b59a14&sxsrf=ANbL-n6qW_s1ov2p-JUzb8lBX_hM-2ECbw%3A1768497888610&source=hp&ei=4CJpafXUI5yzi-gP_tXY4Ac&iflsig=AFdpzrgAAAAAaWkw8E8wYLwh1iURDenMKQfHKOYRIK5S&ved=0ahUKEwj1xL2DiI6SAxWc2QIHHf4qFnwQ4dUDCB4&uact=5&oq=thfjngjkyk&gs_lp=Egdnd3Mtd2l6Igp0aGZqbmdqa3lrMgUQABjvBTIIEAAYgAQYogQyCBAAGIAEGKIEMggQABiABBiiBDIFEAAY7wVIkwtQlwJYqgdwAXgAkAECmAHZAaABuQqqAQUwLjkuMbgBA8gBAPgBAZgCCaACngioAgrCAg0QIxiABBgnGIoFGOoCwgIHECMYJxjqAsICChAjGIAEGCcYigXCAgoQLhiABBhDGIoFwgIKEAAYgAQYQxiKBcICBRAAGIAEwgILEC4YgAQY0QMYxwHCAgUQLhiABMICBxAAGIAEGArCAgkQABiABBgKGAvCAgsQABiABBgBGAoYC8ICBxAuGIAEGA3CAgkQABiABBgKGA3CAgYQABgNGB7CAgcQABiABBgNwgILEAAYgAQYkgMYigXCAgoQABiABBjJAxgNmAMR8QWwBqhiWbg58JIHAzEuOKAH6UyyBwMwLji4B40IwgcHMC4zLjMuM8gHLoAIAA&sclient=gws-wiz"
	url3 := "https://support.google.com/"
	url4 := "https://domains.google/"
	parsedUrl1, _ := url.Parse(url1)
	parsedUrl2, _ := url.Parse(url2)
	if !isSameOrigin(parsedUrl2, parsedUrl1) {
		t.Errorf("query url test failed: %s, %s", parsedUrl1.String(), parsedUrl2.String())
		return
	}
	parsedUrl3, _ := url.Parse(url3)
	parsedUrl4, _ := url.Parse(url4)
	if !isSameOrigin(parsedUrl3, parsedUrl1) {
		t.Errorf("support subdomain test failed: %s, %s", parsedUrl3.String(), parsedUrl1.String())
		return
	}
	if isSameOrigin(parsedUrl4, parsedUrl1) {
		t.Errorf("google domain subdomain test failed: %s, %s", parsedUrl4.String(), parsedUrl1.String())
	}
}