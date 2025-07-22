package util

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"tikdl-web/util/networking"

	"tikdl-web/models"

	"go.uber.org/zap"

	"github.com/aki237/nscjar"
)

var cookiesCache = make(map[string][]*http.Cookie)

func GetExtractorCookies(extractor *models.Extractor) []*http.Cookie {
	if extractor == nil {
		return nil
	}
	cookieFile := extractor.CodeName + ".txt"
	cookies, err := ParseCookieFile(cookieFile)
	if err != nil {
		return nil
	}
	return cookies
}

func ParseCookieFile(fileName string) ([]*http.Cookie, error) {
	cachedCookies, ok := cookiesCache[fileName]
	if ok {
		return cachedCookies, nil
	}
	cookiePath := filepath.Join("cookies", fileName)
	cookieFile, err := os.Open(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie file: %w", err)
	}
	defer cookieFile.Close()

	var parser nscjar.Parser
	cookies, err := parser.Unmarshal(cookieFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cookie file: %w", err)
	}
	cookiesCache[fileName] = cookies
	return cookies, nil
}

func GetLocationURL(
	client models.HTTPClient,
	url string,
	headers map[string]string,
	cookies []*http.Cookie,
) (string, error) {
	if client == nil {
		client = networking.GetDefaultHTTPClient()
	}
	setupRequest := func(method, url string) (*http.Request, error) {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", ChromeUA)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		return req, nil
	}

	// try HEAD first
	req, err := setupRequest(http.MethodHead, url)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		url := resp.Request.URL.String()
		zap.S().Debugf("head response url: %s", url)
		return url, nil
	}

	// fallback to GET
	req, err = setupRequest(http.MethodGet, url)
	if err != nil {
		return "", err
	}
	resp, err = client.Do(req)
	if err == nil {
		resp.Body.Close()
		url := resp.Request.URL.String()
		zap.S().Debugf("get response url: %s", url)
		return url, nil
	}
	return "", errors.New("failed to get location url")
}

func FetchPage(
	client models.HTTPClient,
	method string,
	url string,
	body io.Reader,
	headers map[string]string,
	cookies []*http.Cookie,
) (*http.Response, error) {
	if client == nil {
		client = networking.GetDefaultHTTPClient()
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", ChromeUA)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp, nil
}
