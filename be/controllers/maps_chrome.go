package controllers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chromedp/chromedp"
)

func resolveEnvChromePath(envName string) (string, error) {
	p := strings.TrimSpace(os.Getenv(envName))
	if p == "" {
		return "", nil
	}
	if abs, err := exec.LookPath(p); err == nil {
		return abs, nil
	}
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s tidak valid atau tidak ditemukan: %s", envName, p)
}

func resolveChromeExecPath() (string, error) {
	if p, err := resolveEnvChromePath("CHROME_PATH"); err != nil {
		return "", err
	} else if p != "" {
		return p, nil
	}
	for _, envName := range []string{"CHROME_BIN", "GOOGLE_CHROME_BIN", "PUPPETEER_EXECUTABLE_PATH"} {
		if p, err := resolveEnvChromePath(envName); err != nil {
			return "", err
		} else if p != "" {
			return p, nil
		}
	}
	switch runtime.GOOS {
	case "windows":
		candidates := []string{`C:\Program Files\Google\Chrome\Application\chrome.exe`}
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles(x86)")} {
			if base == "" {
				continue
			}
			candidates = append(candidates, filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates, filepath.Join(local, "Google", "Chrome", "Application", "chrome.exe"))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
		return "", fmt.Errorf("Google Chrome tidak ditemukan; pasang Chrome atau set CHROME_PATH ke chrome.exe")
	case "darwin":
		p := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("Google Chrome tidak ditemukan di %s", p)
	default:
		for _, bin := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
			if p, err := exec.LookPath(bin); err == nil {
				return p, nil
			}
		}
		for _, p := range []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/opt/google/chrome/chrome",
		} {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
		return "", fmt.Errorf("browser Chromium/Chrome tidak ditemukan; set CHROME_PATH atau CHROME_BIN (contoh Railway: /usr/bin/chromium)")
	}
}

// NewGoogleMapsScraper headless=true: Chrome tanpa jendela (untuk API/server). false: tampilan normal (GUI/CLI).
func NewGoogleMapsScraper(headless bool) (*GoogleMapsScraper, error) {
	chromePath, err := resolveChromeExecPath()
	if err != nil {
		return nil, err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	if headless {
		opts = append(opts,
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.WindowSize(1920, 1080),
		)
	} else {
		opts = append(opts,
			chromedp.Flag("headless", false),
			chromedp.Flag("disable-gpu", false),
			chromedp.Flag("disable-dev-shm-usage", false),
		)
	}

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(chromedpLogf))

	return &GoogleMapsScraper{
		ctx:            ctx,
		cancel:         cancel,
		processedNames: make(map[string]bool),
		processedIDs:   make(map[string]bool),
	}, nil
}

func (g *GoogleMapsScraper) Init() error {
	g.progressLine("🚀 Starting browser...")

	err := chromedp.Run(g.ctx,
		chromedp.Navigate("about:blank"),
	)

	return err
}

func (g *GoogleMapsScraper) Close() {
	if g.cancel != nil {
		g.cancel()
	}
	g.progressLine("🔒 Browser closed")
}
