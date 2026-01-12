package devbrowser

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

type PageEntry struct {
	Name     string
	TargetID string
}

type BrowserHost struct {
	profile  string
	headless bool
	cdpPort  int

	mu       sync.Mutex
	pw       *playwright.Playwright
	context  playwright.BrowserContext
	ws       string
	registry map[string]pageHolder
	userData string
	logs     *consoleStore
}

type pageHolder struct {
	page          playwright.Page
	targetID      string
	consoleHooked bool
}

func NewBrowserHost(profile string, headless bool, cdpPort int) *BrowserHost {
	stateBase := filepath.Join(PlatformStateDir(), cacheSubdir, profile)
	return &BrowserHost{
		profile:  profile,
		headless: headless,
		cdpPort:  cdpPort,
		registry: make(map[string]pageHolder),
		userData: filepath.Join(stateBase, "chromium-profile"),
		logs:     newConsoleStore(0),
	}
}

func (b *BrowserHost) WSEndpoint() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ws == "" {
		return "", errors.New("host not started")
	}
	return b.ws, nil
}

func (b *BrowserHost) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.startLocked()
}

func (b *BrowserHost) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			_ = holder.page.Close()
		}
		delete(b.registry, name)
	}

	if b.context != nil {
		_ = b.context.Close()
	}
	b.context = nil

	if b.pw != nil {
		_ = b.pw.Stop()
	}
	b.pw = nil
	b.ws = ""
	if b.logs != nil {
		b.logs.clearAll()
	}
}

func (b *BrowserHost) ListPages() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	names := []string{}
	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			names = append(names, name)
		}
	}
	return names
}

func (b *BrowserHost) ClosePage(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	holder, ok := b.registry[name]
	if !ok {
		return false
	}
	if holder.page != nil && !holder.page.IsClosed() {
		_ = holder.page.Close()
	}
	delete(b.registry, name)
	if b.logs != nil {
		b.logs.clear(name)
	}
	return true
}

func (b *BrowserHost) GetOrCreatePage(name string) (PageEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.context == nil {
		if err := b.startLocked(); err != nil {
			return PageEntry{}, err
		}
	}

	if holder, ok := b.registry[name]; ok && holder.page != nil && !holder.page.IsClosed() {
		if !holder.consoleHooked {
			b.attachConsoleLocked(name, holder.page)
		}
		return PageEntry{Name: name, TargetID: holder.targetID}, nil
	}

	page, err := b.context.NewPage()
	if err != nil {
		return PageEntry{}, err
	}
	tid, err := resolveTargetID(b.context, page)
	if err != nil {
		_ = page.Close()
		return PageEntry{}, err
	}
	b.registry[name] = pageHolder{page: page, targetID: tid}
	b.attachConsoleLocked(name, page)
	return PageEntry{Name: name, TargetID: tid}, nil
}

func (b *BrowserHost) startLocked() error {
	if b.context != nil {
		return nil
	}

	if err := os.MkdirAll(b.userData, 0o755); err != nil {
		return err
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("start playwright: %w", err)
	}

	window, err := WindowSizeFromEnv()
	if err != nil {
		pw.Stop()
		return err
	}

	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		AcceptDownloads:   playwright.Bool(true),
		Headless:          playwright.Bool(b.headless),
		IgnoreHttpsErrors: playwright.Bool(true),
		Args:              ChromiumLaunchArgs(b.cdpPort, window),
	}
	if window != nil {
		opts.Viewport = &playwright.Size{Width: window.Width, Height: window.Height}
		opts.Screen = &playwright.Size{Width: window.Width, Height: window.Height}
	}

	context, err := pw.Chromium.LaunchPersistentContext(b.userData, opts)
	if err != nil {
		pw.Stop()
		return fmt.Errorf("launch context: %w", err)
	}
	context.SetDefaultTimeout(15_000)

	ws, err := waitForWSEndpoint(b.cdpPort, 10*time.Second)
	if err != nil {
		context.Close()
		pw.Stop()
		return err
	}

	pages := context.Pages()
	if len(pages) == 0 {
		p, err := context.NewPage()
		if err != nil {
			context.Close()
			pw.Stop()
			return err
		}
		pages = append(pages, p)
	}

	mainPage := pages[0]
	tid, err := resolveTargetID(context, mainPage)
	if err != nil {
		context.Close()
		pw.Stop()
		return err
	}

	b.pw = pw
	b.context = context
	b.ws = ws
	b.registry["main"] = pageHolder{page: mainPage, targetID: tid}
	b.attachConsoleLocked("main", mainPage)

	for _, pg := range pages[1:] {
		_ = pg.Close()
	}
	return nil
}

func (b *BrowserHost) attachConsoleLocked(name string, page playwright.Page) {
	holder, ok := b.registry[name]
	if !ok {
		// Page must be in registry before attaching console
		return
	}
	if ok && holder.consoleHooked {
		return
	}
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if b.logs != nil {
			b.logs.append(name, msg)
		}
	})
	page.OnPageError(func(err error) {
		if b.logs != nil {
			b.logs.appendPageError(name, err)
		}
	})
	holder.page = page
	holder.consoleHooked = true
	b.registry[name] = holder
}

func (b *BrowserHost) ConsoleLogs(name string, since int64, limit int) ([]ConsoleEntry, int64, error) {
	if since < 0 {
		return nil, 0, errors.New("since must be >= 0")
	}
	if limit < 0 {
		return nil, 0, errors.New("limit must be >= 0")
	}
	b.mu.Lock()
	holder, ok := b.registry[name]
	pageOk := ok && holder.page != nil && !holder.page.IsClosed()
	b.mu.Unlock()
	if !pageOk {
		return nil, 0, errors.New("page not found")
	}
	if b.logs == nil {
		return nil, 0, nil
	}
	entries, lastID := b.logs.list(name, since, limit)
	return entries, lastID, nil
}

func resolveTargetID(context playwright.BrowserContext, page playwright.Page) (string, error) {
	session, err := context.NewCDPSession(page)
	if err != nil {
		return "", err
	}
	infoRaw, err := session.Send("Target.getTargetInfo", map[string]interface{}{})
	_ = session.Detach()
	if err != nil {
		return "", err
	}
	infoMap, ok := infoRaw.(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected target info")
	}
	ti, _ := infoMap["targetInfo"].(map[string]interface{})
	tid, _ := ti["targetId"].(string)
	if tid == "" {
		return "", errors.New("targetId missing")
	}
	return tid, nil
}

func waitForWSEndpoint(port int, timeout time.Duration) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var data struct {
				WSEndpoint string `json:"webSocketDebuggerUrl"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				_ = resp.Body.Close()
				if strings.TrimSpace(data.WSEndpoint) != "" {
					return data.WSEndpoint, nil
				}
			}
			_ = resp.Body.Close()
		} else if err != nil {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		return "", fmt.Errorf("wait ws endpoint: %w", lastErr)
	}
	return "", fmt.Errorf("timed out waiting for Chromium CDP endpoint at %s", url)
}
