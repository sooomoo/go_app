package scrapers

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

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
