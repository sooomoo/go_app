package scrapers

import (
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	firstRunScript = `
		// 消除window.chrome属性差异
        if (!window.chrome) {
            window.chrome = { runtime: {}, version: "CHROME_VERSION" };
        }
        // 移除Headless模式下的webdriver标识
        delete navigator.webdriver;
        // 伪装浏览器语言和时区（与真实环境一致）
        Object.defineProperty(navigator, "language", { get: () => "zh-CN" });
        Object.defineProperty(navigator, "timezoneOffset", { get: () => -480 });
		// 伪装浏览器插件列表
		Object.defineProperty(navigator, "plugins", {
			get: () => [
				{ name: "Chrome PDF Viewer", filename: "internal-pdf-viewer" },
				{ name: "Shockwave Flash", filename: "pepflashplayer.dll" }
			]
		});
		// 修复mimeTypes（与插件对应）
		Object.defineProperty(navigator, "mimeTypes", {
			get: () => [
				{ type: "application/pdf", suffixes: "pdf", enabledPlugin: navigator.plugins[0] },
				{ type: "application/x-shockwave-flash", suffixes: "swf", enabledPlugin: navigator.plugins[1] }
			]
		});
	`
)

type userAgentConfig struct {
	userAgent       string
	chromeVersion   string
	secChUa         string
	secChUaPlatform string
	acceptEncoding  string
	acceptLanguage  string
}

// 第一个为默认的 user-agent
var userAgentsConfigs = []userAgentConfig{
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
		chromeVersion:   "140.0.0.0",
		secChUa:         `"Google Chrome";v="140", "Not?A_Brand";v="8", "Chromium";v="140"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
	},
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		chromeVersion:   "141.0.0.0",
		secChUa:         `"Google Chrome";v="141", "Not?A_Brand";v="8", "Chromium";v="141"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
	},
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0",
		chromeVersion:   "139.0.0.0",
		secChUa:         `"Microsoft Edge";v="139", "Not?A_Brand";v="8", "Chromium";v="139"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zsdch, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
	},
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0",
		chromeVersion:   "141.0.0.0",
		secChUa:         `"Microsoft Edge";v="141", "Not?A_Brand";v="8", "Chromium";v="141"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zsdch, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
	},
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		chromeVersion:   "123.0.0.0",
		secChUa:         `"Google Chrome";v="123", "Not?A_Brand";v="8", "Chromium";v="123"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
	},
	// {
	// 	userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
	// 	secChUa:         `"Firefox";v="124", "Not?A_Brand";v="8", "Chromium";v="124"`,
	// 	secChUaPlatform: `"Windows"`,
	// },
	{
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.2420.81",
		chromeVersion:   "123.0.2420.81",
		secChUa:         `"Microsoft Edge";v="123","Not?A_Brand";v="8","Chromium";v="123"`,
		secChUaPlatform: `"Windows"`,
		acceptEncoding:  "gzip, deflate, br, zsdch, zstd",
		acceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
	},
}

// 随机返回一个userAgent
func randomUserAgentConfig() userAgentConfig {
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := localRand.Intn(len(userAgentsConfigs))
	if index < 0 || index >= len(userAgentsConfigs) {
		return userAgentsConfigs[0]
	}
	return userAgentsConfigs[index]
}

type linkLockData struct {
	sync.RWMutex
	lastVisitTime time.Time
}

var linkLockManager = map[string]*linkLockData{}
var linkLockManagerLock = sync.RWMutex{}

func getLinkLock(host string) *linkLockData {
	linkLockManagerLock.Lock()
	defer linkLockManagerLock.Unlock()
	linkLock, ok := linkLockManager[host]
	if !ok {
		linkLock = &linkLockData{
			RWMutex:       sync.RWMutex{},
			lastVisitTime: time.Now(),
		}
		linkLockManager[host] = linkLock
	}
	return linkLock
}

// 验证访问指定的链接前，是否需要等待；如果不等待，可能触发目标网站的限流机制，导致访问失败
func waitIfNeededBeforeVisit(link string) {
	p, err := url.Parse(link)
	if err != nil {
		return
	}
	host := strings.TrimSpace(strings.ToLower(p.Host))
	if len(host) == 0 {
		return
	}

	linkLock := getLinkLock(host)
	linkLock.Lock()
	defer linkLock.Unlock()

	diff := time.Since(linkLock.lastVisitTime)
	if diff < 3*time.Second {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		// 生成一个 [2, 5] 的随机整数
		min := 2
		max := 5
		randomNum2 := localRand.Intn(max-min+1) + min
		sleepDur := time.Duration(randomNum2) * time.Second
		time.Sleep(sleepDur)
		linkLock.lastVisitTime = time.Now()
	}
}
