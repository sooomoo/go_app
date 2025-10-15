package scrapers

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
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
	if diff < 5*time.Second {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		// 生成一个 [5, 10] 的随机整数
		min := 5
		max := 10
		randomNum2 := localRand.Intn(max-min+1) + min
		sleepDur := time.Duration(randomNum2) * time.Second
		time.Sleep(sleepDur)
		linkLock.lastVisitTime = time.Now()
	}
}

type SiteScraper struct {
	timeout                 time.Duration
	options                 []chromedp.ExecAllocatorOption
	enableLifecycleEventLog bool
}

// NewSiteScraper 创建一个LinkScraper实例
func NewSiteScraper(timeout time.Duration, headless bool) *SiteScraper {
	options := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun, //设置网站不是首次运行
		chromedp.NoDefaultBrowserCheck,
		// chromedp.IgnoreCertErrors,
		chromedp.DisableGPU,
		chromedp.Flag("blink-settings", "imagesEnabled=false"), //开启图像界面,重点是开启这个
		chromedp.Flag("ignore-certificate-errors", true),       //忽略错误
		chromedp.Flag("disable-web-security", true),            //禁用网络安全标志
		chromedp.Flag("disable-extensions", true),              //开启插件支持
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("incognito", true),
		chromedp.Flag("disable-infobars", true),

		// After Puppeteer's default behavior.
		chromedp.Flag("disable-background-networking", false),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
	}
	if headless {
		options = append(options, chromedp.Headless)
	}

	s := &SiteScraper{
		timeout:                 timeout,
		options:                 options,
		enableLifecycleEventLog: false,
	}

	return s
}

// NewLinkScraperHeadless 创建一个无头浏览器的LinkScraper实例
func NewLinkScraperHeadless(timeout time.Duration) *SiteScraper {
	return NewSiteScraper(timeout, true)
}

func (s *SiteScraper) EnableLifecycleEventLog() {
	s.enableLifecycleEventLog = true
}

func (s *SiteScraper) DisableLifecycleEventLog() {
	s.enableLifecycleEventLog = false
}

// 记录日志的逻辑
func (s *SiteScraper) printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// RunActionsOnPage 执行页面操作
//
// 1. 等待页面加载完成（或执行超时）
//
// 2. 执行actions
func (s *SiteScraper) RunActionsOnPage(link string, actions ...chromedp.Action) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("RunActionsOnPage panic: %v\n", err)
		}
	}()

	link = strings.TrimSpace(link)
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}

	waitIfNeededBeforeVisit(link)

	agent := randomUserAgentConfig()

	options := append([]chromedp.ExecAllocatorOption{}, s.options...)
	options = append(options, chromedp.UserAgent(agent.userAgent)) // 使用随机的user-agent
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer allocCancel()
	// 创建上下文实例
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(s.printf))
	defer ctxCancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, s.timeout) // 创建超时上下文
	defer timeoutCancel()

	loadCount := atomic.Int32{}
	hasInit := atomic.Bool{}

	chromedp.ListenTarget(ctx, func(ev any) {
		switch ev := ev.(type) {
		case *page.EventLifecycleEvent:
			if s.enableLifecycleEventLog {
				fmt.Printf("LifecycleEvent: %s\n", ev.Name)
			}
			// commit, DOMContentLoaded, load, networkAlmostIdle, networkIdle,
			if strings.EqualFold(ev.Name, "InteractiveTime") {
				if hasInit.Load() {
					loadCount.Add(1)
				}
			} else if strings.EqualFold(ev.Name, "firstMeaningfulPaint") || strings.EqualFold(ev.Name, "firstContentfulPaint") {
				hasInit.Store(true)
			}
		default:
			return
		}
	})
	waitForLoadAndIdle := func(c context.Context) error {
		ctx, timeoutCancel := context.WithTimeout(c, s.timeout) // 创建超时上下文
		defer timeoutCancel()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if loadCount.Load() > 0 {
					return nil
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}

	firstScript := strings.ReplaceAll(firstRunScript, "CHROME_VERSION", agent.chromeVersion)
	firstRunAction := chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(firstScript, nil).Do(ctx)
	})

	tasks := chromedp.Tasks{
		// network.Enable(),
		firstRunAction,
		chromedp.Navigate(link),
		firstRunAction,
		chromedp.ActionFunc(waitForLoadAndIdle),
	}
	if len(actions) > 0 {
		tasks = append(tasks, actions...)
	}
	err := chromedp.Run(ctx, tasks)

	chromedp.Cancel(ctx)

	return err
}

// GetPageTitleContent 获取页面标题和内容(html)
//
// 可抓取同源的内嵌 iframe 页面
func (s *SiteScraper) GetPageHtml(link string) (string, string, error) {
	html, title, ogTitle, twitterTitle, iframes := "", "", "", "", ""
	err := s.RunActionsOnPage(link,
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.Evaluate(`document.querySelector('meta[property="og:title"]')?.content || ''`, &ogTitle, chromedp.EvalIgnoreExceptions),
		chromedp.Evaluate(`document.querySelector('meta[property="twitter:title"]')?.content || ''`, &twitterTitle, chromedp.EvalIgnoreExceptions),
		chromedp.Evaluate(`
			function getIframeText() {
				try {
					const arr = []
					const iframesQuery = document.querySelectorAll('iframe');
					if (iframesQuery.length === 0) return '';
					iframesQuery.forEach(iframe => {
						try {
							const id = iframe.id || ''
							if (id.includes("_ads_")) return;
							if (iframe.style && iframe.style.display === 'none') return;
							if (iframe.style && iframe.style.visibility === 'hidden') return;
							const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
							if (iframeDoc) {
								const textContent =( iframeDoc.body.innerText || iframeDoc.body.textContent).trim();
								if (textContent) arr.push(textContent)
							}
						} catch (e) {
							console.log(e)
						}
					})
					return arr.join('\n\n')
				} catch (e) {
					console.log(e)
				}
				return ''
			}
			getIframeText()
		`, &iframes, chromedp.EvalIgnoreExceptions),
	)
	if err != nil {
		return "", "", err
	}

	iframes = strings.TrimSpace(iframes)
	if len(iframes) > 0 {
		html = html + "\n\n" + iframes
	}

	ogTitle = strings.TrimSpace(ogTitle)
	twitterTitle = strings.TrimSpace(twitterTitle)
	if len(ogTitle) > 0 {
		title = ogTitle
	} else if len(twitterTitle) > 0 {
		title = twitterTitle
	}

	return strings.TrimSpace(title), strings.TrimSpace(html), nil
}
