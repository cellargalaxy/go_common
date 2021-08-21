package model

type HttpRequestParam struct {
	Url    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}
