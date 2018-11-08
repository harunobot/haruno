package clients

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// HTTPClient 支持socks或者http代理的，以及cookie的http客户端
type HTTPClient struct {
	http.Client
	Header http.Header
}

// DefaultHTTPClient 默认的公共http客户端
// 建议没有特殊需求的功能都使用这个客户端
var DefaultHTTPClient = NewHTTPClient()

// NewHTTPClient 创建新的 http client 客户端
func NewHTTPClient() *HTTPClient {
	client := new(HTTPClient)
	// 设置默认的请求头
	client.Header = make(http.Header)
	client.Header.Set("User-Agent", "Haruno Robot")
	jar, _ := cookiejar.New(nil)
	client.Jar = jar
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	client.Transport = transport
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

// Head 增强http.Client.Head方法
func (c *HTTPClient) Head(url string) (*http.Response, error) {
	req, err := c.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Get 增强http.Client.Get方法
func (c *HTTPClient) Get(url string) (*http.Response, error) {
	req, err := c.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post 增强http.Client.Post方法
func (c *HTTPClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// PostForm 增强http.Client.PostForm方法
func (c *HTTPClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}
