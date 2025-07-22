package models

type EdgeProxyResponse struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Text       string            `json:"text"`
	Headers    map[string]string `json:"headers"`
	Cookies    []string          `json:"cookies"`
}
