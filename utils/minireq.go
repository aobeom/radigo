package utils

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// MiniRequestBasic 提供基本 HTTP 请求
type MiniRequestBasic struct{}

// Minireq 初始化
var Minireq *MiniRequestBasic

// Proxy 设置代理
var Proxy = ""

func init() {
	Minireq = NewMiniReq()
}

// NewMiniReq 初始化 NewMiniReq
func NewMiniReq() (n *MiniRequestBasic) {
	n = new(MiniRequestBasic)
	return
}

// s5Proxy 设置 socket5 代理
func s5Proxy(proxyURL string) (transport *http.Transport) {
	dialer, err := proxy.SOCKS5("tcp", proxyURL,
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		log.Fatal(" [S5 Proxy Error]: ", err)
	}
	transport = &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return
}

// GetRes 获取 Response
func (mr *MiniRequestBasic) GetRes(url string, headers http.Header, params map[string]string) (res *http.Response) {
	httpClient := http.Client{Timeout: 30 * time.Second}
	if Proxy != "" {
		transport := s5Proxy(Proxy)
		httpClient = http.Client{Timeout: 30 * time.Second, Transport: transport}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Panic(" [GetRes - Request Error]: ", err)
	}
	req.Header = headers
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	res, err = httpClient.Do(req)

	if err != nil {
		log.Panic(" [GetRes - Response Error]: ", err)
	}
	return
}

// GetBody Http Get 获取 Body 内容
func (mr *MiniRequestBasic) GetBody(url string, headers http.Header, params map[string]string) []byte {
	httpClient := http.Client{Timeout: 30 * time.Second}
	if Proxy != "" {
		transport := s5Proxy(Proxy)
		httpClient = http.Client{Timeout: 30 * time.Second, Transport: transport}
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Panic(" [GetBody - Request Error]: ", err)
	}

	req.Header = headers

	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	res, err := httpClient.Do(req)
	if err != nil {
		log.Panic(" [GetBody - Response Error]: ", err)
	}

	if res.StatusCode != 200 {
		log.Panic(" [GetBody - Response Code != 200]: ", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic(" [GetBody - Body Error]: ", err)
	}
	return body
}

// PostBody Http Post 获取 Body 内容
func (mr *MiniRequestBasic) PostBody(url string, headers http.Header, reader io.Reader) []byte {
	httpClient := http.Client{Timeout: 30 * time.Second}
	if Proxy != "" {
		transport := s5Proxy(Proxy)
		httpClient = http.Client{Timeout: 30 * time.Second, Transport: transport}
	}
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		log.Panic(" [Post - Request Error]: ", err)
	}

	username := ""
	password := ""

	req.Header = headers

	if headers.Get("username") != "" {
		username = headers.Get("username")
		req.Header.Del("username")
	}

	if headers.Get("password") != "" {
		password = headers.Get("password")
		req.Header.Del("password")
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	res, err := httpClient.Do(req)

	if err != nil {
		log.Panic(" [Post - Response Error]: ", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic(" [Get - Body Error]: ", err)
	}
	return body
}
