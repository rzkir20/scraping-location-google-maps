package controllers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"location/types"

	"github.com/chromedp/chromedp"
)

func evalResultToString(ctx context.Context, script string) (string, error) {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &raw))
	if err != nil {
		return "", err
	}
	var v interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return extractString(v), nil
}

func extractString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]interface{}); ok {
		if res, ok := m["result"].(map[string]interface{}); ok {
			if val, ok := res["value"].(string); ok {
				return val
			}
			if val, ok := res["Value"].(string); ok {
				return val
			}
			if val := res["value"]; val != nil && fmt.Sprint(val) != "map[]" {
				return fmt.Sprint(val)
			}
		}
		if val, ok := m["value"].(string); ok {
			return val
		}
		if val, ok := m["Value"].(string); ok {
			return val
		}
		if val := m["value"]; val != nil {
			s := fmt.Sprint(val)
			if s != "" && s != "map[]" {
				return s
			}
		}
		if val := m["Value"]; val != nil {
			s := fmt.Sprint(val)
			if s != "" && s != "map[]" {
				return s
			}
		}
		return ""
	}
	return fmt.Sprint(v)
}

func evalResultToBool(ctx context.Context, script string) (bool, error) {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &raw))
	if err != nil {
		return false, err
	}
	var v interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	if b, ok := v.(bool); ok {
		return b, nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		if val, ok := m["value"].(bool); ok {
			return val, nil
		}
	}
	return false, nil
}

func chromedpLogf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "could not unmarshal event:") {
		return
	}
	log.Print(msg)
}

type GoogleMapsScraper struct {
	ctx            context.Context
	cancel         context.CancelFunc
	processedNames map[string]bool
	processedIDs   map[string]bool
	mu             sync.Mutex
}

func NewGoogleMapsScraper() *GoogleMapsScraper {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("disable-dev-shm-usage", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(chromedpLogf))

	return &GoogleMapsScraper{
		ctx:            ctx,
		cancel:         cancel,
		processedNames: make(map[string]bool),
		processedIDs:   make(map[string]bool),
	}
}

func (g *GoogleMapsScraper) Init() error {
	log.Println("🚀 Starting browser...")

	err := chromedp.Run(g.ctx,
		chromedp.Navigate("about:blank"),
	)

	return err
}

func (g *GoogleMapsScraper) ScrapeCoffeeShops(url string, maxResults int) ([]types.StoreInfo, error) {
	log.Printf("📍 Navigating to: %s\n", url)
	log.Println("📋 Hanya ambil: Nama + Telepon (yang belum punya website)")

	scrapeCtx, cancel := context.WithTimeout(g.ctx, 5*time.Minute)
	defer cancel()

	err := chromedp.Run(scrapeCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	log.Println("⏳ Waiting for search results to load...")
	err = chromedp.Run(scrapeCtx,
		chromedp.Sleep(8*time.Second),
	)
	if err != nil {
		log.Printf("⚠️  Warning waiting: %v\n", err)
	}

	err = chromedp.Run(scrapeCtx,
		chromedp.WaitVisible("[role=\"main\"]", chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("⚠️  Sidebar not found, continuing anyway...\n")
	}

	err = chromedp.Run(scrapeCtx,
		chromedp.Sleep(1*time.Second),
	)

	var pageInfoRaw interface{}
	chromedp.Run(scrapeCtx,
		chromedp.Evaluate(`
			() => {
				return {
					url: window.location.href,
					title: document.title,
					hasSidebar: !!document.querySelector('[role="main"]'),
					hasResults: document.querySelectorAll('div[data-result-index]').length > 0
				};
			}
		`, &pageInfoRaw),
	)
	log.Printf("📄 Page info: %v\n", pageInfoRaw)

	stores := []types.StoreInfo{}
	scrollAttempts := 0
	maxScrollAttempts := 15

	log.Println("🔍 Scrolling dan mengumpulkan data...")

	for len(stores) < maxResults && scrollAttempts < maxScrollAttempts {
		chromedp.Run(scrapeCtx,
			chromedp.Evaluate(`(function(){
				var s = document.querySelector('[role="main"]') || document.querySelector('.m6QErb');
				if(s) s.scrollTop = s.scrollHeight;
				return true;
			})()`, nil),
		)
		time.Sleep(400 * time.Millisecond)

		cardCountInt := g.getCardCount(scrapeCtx)
		if cardCountInt == 0 {
			scrollAttempts++
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("🔎 %d kartu\n", cardCountInt)

		for i := 0; i < cardCountInt && len(stores) < maxResults; i++ {
			cardCtx, cancel := context.WithTimeout(scrapeCtx, 12*time.Second)
			store, err := g.processCard(i, cardCtx)
			cancel()
			if err != nil {
				log.Printf("❌ Kartu %d: %v\n", i, err)
				continue
			}
			if store != nil {
				if store.HasWebsite {
					log.Printf("⏭️  Lewati %s (punya website, tidak dimasukkan)\n", store.Name)
					continue
				}
				stores = append(stores, *store)
				log.Printf("✅ [%d] %s - %s\n", len(stores), store.Name, getPhoneDisplay(store.Phone))
			}
		}

		scrollAttempts++
		log.Printf("📊 %d/%d (scroll %d/%d)\n", len(stores), maxResults, scrollAttempts, maxScrollAttempts)
	}

	return stores, nil
}

func (g *GoogleMapsScraper) getCardCount(ctx context.Context) int {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		(function(){
			var cards = document.querySelectorAll('a[href*="/maps/place/"]');
			if (cards.length === 0) cards = document.querySelectorAll('div[data-result-index]');
			if (cards.length === 0) cards = document.querySelectorAll('[data-result-index]');
			return cards.length;
		})()
	`, &raw))
	if err != nil {
		return 0
	}
	var v interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func (g *GoogleMapsScraper) processCard(index int, ctx context.Context) (*types.StoreInfo, error) {
	clickJS := fmt.Sprintf(`
		(function(){
			var cards = document.querySelectorAll('a[href*="/maps/place/"]');
			if (cards.length === 0) cards = document.querySelectorAll('div[data-result-index]');
			if (cards.length === 0) cards = document.querySelectorAll('[data-result-index]');
			var card = cards[%d];
			if (!card) return false;
			card.scrollIntoView({ block: 'center' });
			card.click();
			return true;
		})()
	`, index)
	var clicked interface{}
	if err := chromedp.Run(ctx, chromedp.Evaluate(clickJS, &clicked)); err != nil {
		return nil, err
	}
	if b, ok := clicked.(bool); ok && !b {
		return nil, nil
	}

	panelCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	err := chromedp.Run(panelCtx,
		chromedp.WaitVisible("h1.DUwDvf", chromedp.ByQuery),
	)
	cancel()
	if err != nil {
		g.closeDetailPanel(ctx)
		return nil, fmt.Errorf("panel tidak muncul")
	}
	time.Sleep(400 * time.Millisecond)

	getDataJS := `
		(function(){
			var name = '';
			var el = document.querySelector('h1.DUwDvf') || document.querySelector('h1[data-value="name"]') || document.querySelector('h1');
			if (el) name = (el.textContent || '').trim();
			var phone = '';
			var tel = document.querySelector('a[href^="tel:"]');
			if (tel) {
				phone = (tel.getAttribute('href') || '').replace(/^tel:\\s*/i,'').trim();
			} else {
				var btn = document.querySelector('button[data-item-id*="phone"]');
				if (btn) phone = (btn.getAttribute('aria-label') || btn.textContent || '').replace(/[^0-9+\\-\\s()]/g,'').trim();
			}
			var hasWebsite = false;
			var links = document.querySelectorAll('a[href^="http"]');
			for (var i = 0; i < links.length; i++) {
				var h = (links[i].getAttribute('href') || '');
				if (h.indexOf('google.com') < 0 && (h.indexOf('http') === 0 || h.indexOf('www') >= 0)) { hasWebsite = true; break; }
			}
			return JSON.stringify({ name: name, phone: phone, hasWebsite: hasWebsite });
		})()
	`
	var raw []byte
	if err := chromedp.Run(ctx, chromedp.Evaluate(getDataJS, &raw)); err != nil {
		g.closeDetailPanel(ctx)
		return nil, err
	}
	var data struct {
		Name       string `json:"name"`
		Phone      string `json:"phone"`
		HasWebsite bool   `json:"hasWebsite"`
	}
	if len(raw) > 0 {
		var jsonStr string
		if err := json.Unmarshal(raw, &jsonStr); err == nil && jsonStr != "" {
			_ = json.Unmarshal([]byte(jsonStr), &data)
		} else {
			_ = json.Unmarshal(raw, &data)
		}
	}

	g.closeDetailPanel(ctx)

	name := strings.TrimSpace(data.Name)
	if name == "" {
		return nil, nil
	}
	g.mu.Lock()
	if g.processedNames[name] {
		g.mu.Unlock()
		return nil, nil
	}
	g.processedNames[name] = true
	g.mu.Unlock()

	return &types.StoreInfo{
		Name:       name,
		Phone:      strings.TrimSpace(data.Phone),
		HasWebsite: data.HasWebsite,
	}, nil
}

func (g *GoogleMapsScraper) closeDetailPanel(ctx context.Context) {
	chromedp.Run(ctx,
		chromedp.Evaluate(`
			() => {
				const backButton = document.querySelector('button[aria-label*="Back"]') ||
				                  document.querySelector('button[aria-label*="back"]') ||
				                  document.querySelector('button[jsaction*="back"]');
				if (backButton) {
					backButton.click();
					return true;
				}
				return false;
			}
		`, nil),
	)
	time.Sleep(200 * time.Millisecond)
}

func (g *GoogleMapsScraper) SaveToFile(stores []types.StoreInfo, filename string) error {
	if filename == "" {
		filename = "results.json"
	}

	data, err := json.MarshalIndent(stores, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	log.Printf("💾 Results saved to: %s\n", filename)
	return nil
}

func (g *GoogleMapsScraper) SaveToCSV(stores []types.StoreInfo, filename string) error {
	if filename == "" {
		filename = "results.csv"
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"Nama Toko", "Phone Number"}); err != nil {
		return err
	}

	for _, store := range stores {
		phone := store.Phone
		if phone == "" {
			phone = "N/A"
		}
		if err := writer.Write([]string{store.Name, phone}); err != nil {
			return err
		}
	}

	log.Printf("💾 CSV saved to: %s\n", filename)
	return nil
}

func (g *GoogleMapsScraper) Close() {
	if g.cancel != nil {
		g.cancel()
	}
	log.Println("🔒 Browser closed")
}

func getPhoneDisplay(phone string) string {
	if phone == "" {
		return "N/A"
	}
	return phone
}
