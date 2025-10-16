package scrapers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type HttpScraper struct {
	headTimeout time.Duration
	timeout     time.Duration
	header      http.Header
	jar         *cookiejar.Jar
	transport   *http.Transport
	maxRetries  int

	limitReqFrequence bool
}

// NewHttpScraper 创建一个新的 HttpScraper 实例

// 默认 30s 超时, 默认重试2次（加上失败前的第一次请求，共3次请求）
func NewHttpScraper() *HttpScraper {
	jar, _ := cookiejar.New(nil)

	// 创建一个自定义的 Transport，并配置 TLS 设置
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 关键：跳过证书验证
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return nil // 直接返回 nil，忽略验证错误
			},
		},
	}

	timeout := 30 * time.Second
	return &HttpScraper{
		timeout:           timeout,
		headTimeout:       timeout,
		header:            http.Header{},
		jar:               jar,
		limitReqFrequence: true,
		transport:         tr,
		maxRetries:        2,
	}
}

func (s *HttpScraper) SetHeadTimeout(timeout time.Duration) {
	s.headTimeout = timeout
}

func (s *HttpScraper) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
}

// SetLimitReqFrequence 设置是否限制请求频率（仅针对同一链接）
func (s *HttpScraper) SetLimitReqFrequence(limit bool) {
	s.limitReqFrequence = limit
}

func (s *HttpScraper) SetHeader(key, value string) {
	s.header.Set(key, value)
}

func (s *HttpScraper) AddHeader(key, value string) {
	s.header.Add(key, value)
}

func (s *HttpScraper) DoRequest(method string, link string, body io.Reader, timeout time.Duration) (response *http.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("painc:%v", r)
		}
	}()
	if s.limitReqFrequence {
		waitIfNeededBeforeVisit(link)
	}

	var req *http.Request
	req, err = http.NewRequest(method, link, body)
	if err != nil {
		return nil, err
	}

	uaConfig := randomUserAgentConfig()

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("User-Agent", uaConfig.userAgent)
	req.Header.Set("Accept-Encoding", uaConfig.acceptEncoding)
	req.Header.Set("Accept-Language", uaConfig.acceptLanguage)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Ch-Ua", uaConfig.secChUa)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", uaConfig.secChUaPlatform)
	req.Header.Set("Referer", link)

	// 设置Origin
	p, err := url.Parse(link)
	if err == nil {
		referer := fmt.Sprintf("%s://%s/", p.Scheme, p.Host)
		req.Header.Set("Origin", referer)
	}

	if len(s.header) > 0 {
		for k, v := range s.header {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}

	// 3. 创建HTTP客户端
	client := &http.Client{
		Transport: s.transport,
		Timeout:   timeout,
		Jar:       s.jar,
	}

	// 4. 发送请求
	// response, err = client.Do(req)
	maxReqCount := s.maxRetries + 1
	for i := range maxReqCount {
		response, err = client.Do(req)
		if err != nil {
			// 检查是否是连接关闭类错误
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "idle HTTP channel") {
				if i == maxReqCount {
					// 重试多次后仍失败，最终处理
					return nil, fmt.Errorf("after %d retries: %w", maxReqCount, err)
				}
				fmt.Printf("请求失败，进行第%d次重试: %v\n", i+1, err)
				time.Sleep(time.Duration((i+1)*100) * time.Millisecond) // 指数退避更佳
				continue
			} else {
				// 其他错误，直接返回
				return nil, err
			}
		}
		break // 成功则跳出循环
	}
	return
}

func (s *HttpScraper) DoGet(url string) (*http.Response, error) {
	return s.DoRequest(http.MethodGet, url, nil, s.timeout)
}

func (s *HttpScraper) DoPost(url string, body io.Reader) (*http.Response, error) {
	return s.DoRequest(http.MethodPost, url, body, s.timeout)
}

func (s *HttpScraper) DoHead(url string) (http.Header, error) {
	resp, err := s.DoRequest(http.MethodHead, url, nil, s.headTimeout)
	if err != nil {
		return http.Header{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return http.Header{}, fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}

	return resp.Header, nil
}

// 获取链接的MineType
func (s *HttpScraper) GetMineType(link string) (*MineType, error) {
	// 如果HEAD方法失败或不确定，通过内容检测
	mineType, err := s.checkMineTypeByContent(link)
	if err != nil {
		if strings.HasPrefix(err.Error(), "StatusCode:") {
			return nil, err
		} else if strings.Contains(err.Error(), "deadline exceeded") {
			err = errors.New("Timeout")
		}
		return nil, err
	}

	return NewMineType(mineType), nil
}

func (s *HttpScraper) checkMineTypeByContent(url string) (string, error) {
	resp, err := s.DoRequest(http.MethodGet, url, nil, s.timeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if len(contentType) == 0 {
		buffer := make([]byte, 512)
		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		contentType = http.DetectContentType(buffer[:n])
	}

	return contentType, nil
}
