package clients

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// HTTPClient 支持socks或者http代理的，以及cookie的http客户端
type HTTPClient struct {
	http.Client
	Header http.Header
}

// Default 默认的http客户端
// 建议没有特殊需求的功能都使用这个客户端
var Default *HTTPClient

// SetHeader 设置http请求头
func (c *HTTPClient) SetHeader(header http.Header) {
	c.Header = header
}

// Init 初始化cookiejar
func (c *HTTPClient) Init(header *http.Header) {
	if c.Jar == nil {
		jar, _ := cookiejar.New(nil)
		c.Jar = jar
	}
	if header != nil {
		c.SetHeader(*header)
	}
}

// NewHTTPClient 创建新的 http client 客户端
// proxyURL 客户端代理 proxy: socks or http
func NewHTTPClient(proxyURL string) *HTTPClient {
	client := &HTTPClient{}
	if proxyURL != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		}
		transport := &http.Transport{Proxy: proxy}
		client.Transport = transport
	}
	return client
}

// NewRequest 使用客户端创建http请求
func (c *HTTPClient) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Header != nil {
		req.Header = c.Header
	}
	return req, nil
}
