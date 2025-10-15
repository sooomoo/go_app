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

	limitReqFrequence bool
}

// NewHttpScraper 创建一个新的 HttpScraper 实例

// 默认 30s 超时
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

	if len(s.header) > 0 {
		for k, v := range s.header {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}
	// 如果没有设置 Referer，则根据访问地址设置
	if len(strings.TrimSpace(req.Header.Get("Referer"))) == 0 {
		p, err := url.Parse(link)
		if err == nil {
			referer := fmt.Sprintf("%s://%s/", p.Scheme, p.Host)
			req.Header.Set("Referer", referer)
		}
	}

	// 3. 创建HTTP客户端
	client := &http.Client{
		Transport: s.transport,
		Timeout:   timeout,
		Jar:       s.jar,
	}

	// 4. 发送请求
	response, err = client.Do(req)
	return
}

func (s *HttpScraper) DoGet(url string) (*http.Response, error) {
	return s.DoRequest(http.MethodGet, url, nil, s.timeout)
}

func (s *HttpScraper) DoPost(url string, body io.Reader) (*http.Response, error) {
	return s.DoRequest(http.MethodPost, url, body, s.timeout)
}

// 获取链接的MineType
func (s *HttpScraper) GetMineType(link string) (*MineType, error) {
	// 首先尝试通过HEAD请求快速判断
	mineType, err := s.checkMineTypeByHeader(link)
	if err == nil && len(mineType) > 0 {
		return NewMineType(mineType), nil
	}
	// if err != nil {
	// 	if strings.HasPrefix(err.Error(), "StatusCode:") {
	// 		return nil, err
	// 	}
	// 	//  else if strings.Contains(err.Error(), "deadline exceeded") {
	// 	// 	return nil, errors.New("Timeout")
	// 	// }
	// }

	// 如果HEAD方法失败或不确定，通过内容检测
	mineType, err = s.checkMineTypeByContent(link)
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

func (s *HttpScraper) checkMineTypeByHeader(url string) (string, error) {
	resp, err := s.DoRequest(http.MethodHead, url, nil, s.headTimeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	return contentType, nil
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

type MineType struct {
	value string
}

func NewMineType(value string) *MineType {
	segs := strings.Split(value, ";")
	if len(segs) > 0 {
		value = strings.TrimSpace(segs[0])
	}
	return &MineType{value: value}
}

func (c MineType) GetValue() string {
	return c.value
}

func (c MineType) IsPdf() bool {
	return c.value == "application/pdf"
}

func (c MineType) IsHtml() bool {
	return strings.EqualFold(c.value, "text/html")
}

func (c MineType) IsPlainText() bool {
	return strings.EqualFold(c.value, "text/plain")
}

func (c MineType) IsJpeg() bool {
	return strings.EqualFold(c.value, "image/jpeg")
}

func (c MineType) IsPng() bool {
	return strings.EqualFold(c.value, "image/png")
}

func (c MineType) IsGif() bool {
	return strings.EqualFold(c.value, "image/gif")
}

func (c MineType) IsWebp() bool {
	return strings.EqualFold(c.value, "image/webp")
}

func (c MineType) IsBmp() bool {
	return strings.EqualFold(c.value, "image/bmp")
}

// PowerPoint 2007及以后版本基于XML的开放文档格式。这是目前最常用的类型
func (c MineType) IsPPTX() bool {
	return strings.EqualFold(c.value, "application/vnd.openxmlformats-officedocument.presentationml.presentation")
}

// Microsoft PowerPoint 97-2003创建的二进制格式演示文稿
func (c MineType) IsPPT2003() bool {
	return strings.EqualFold(c.value, "application/vnd.ms-powerpoint")
}
