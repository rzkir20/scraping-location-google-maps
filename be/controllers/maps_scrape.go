package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"location/types"

	"github.com/chromedp/chromedp"
)

// dismissBlockingUI menutup overlay cookie/consent yang menghalangi feed hasil.
func (g *GoogleMapsScraper) dismissBlockingUI(ctx context.Context) {
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function(){
				function tryClick(sel) {
					var el = document.querySelector(sel);
					if (el && el.offsetParent !== null) { el.click(); return true; }
					return false;
				}
				var sels = [
					'button#L2AGLb',
					'button[aria-label="Accept all"]',
					'button[aria-label*="Accept all" i]',
					'form[action*="consent"] button',
					'button[jsname="hZCF7c"]'
				];
				for (var i = 0; i < sels.length; i++) {
					if (tryClick(sels[i])) return true;
				}
				var btns = document.querySelectorAll('button');
				for (var j = 0; j < btns.length; j++) {
					var t = (btns[j].textContent || '').trim();
					if (/^accept all$/i.test(t) || /^setuju$/i.test(t)) {
						if (btns[j].offsetParent !== null) { btns[j].click(); return true; }
					}
				}
				return false;
			})()
		`, nil),
	)
	time.Sleep(350 * time.Millisecond)
}

func (g *GoogleMapsScraper) ScrapeCoffeeShops(url string, maxResults int) ([]types.StoreInfo, ScrapeSummary, error) {
	summary := ScrapeSummary{TargetMax: maxResults}
	g.reportProgress(0, maxResults)
	g.progressf("📍 Navigating to: %s", url)
	g.progressf("📋 Target: kumpulkan hingga %d listing pertama yang terbaca.", maxResults)

	// Banyak kartu × buka panel butuh waktu; per kartu ~sedikit lebih ketat karena skip URL sudah diproses.
	scrapeBudget := 10*time.Minute + time.Duration(maxResults)*18*time.Second
	scrapeCtx, cancel := context.WithTimeout(g.ctx, scrapeBudget)
	defer cancel()

	err := chromedp.Run(scrapeCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(1200*time.Millisecond),
	)
	if err != nil {
		return nil, summary, fmt.Errorf("failed to navigate: %w", err)
	}
	g.lastSearchURL = url

	g.dismissBlockingUI(scrapeCtx)

	g.progressLine("⏳ Waiting for search results to load...")
	err = chromedp.Run(scrapeCtx,
		chromedp.Poll(
			`(function(){
				var feed = document.querySelector('div[role="feed"]');
				if (feed && feed.querySelector('a[href*="/maps/place/"]')) return true;
				if (document.querySelectorAll('a[href*="/maps/place/"]').length > 0) return true;
				return false;
			})()`,
			nil,
			chromedp.WithPollingTimeout(45*time.Second),
			chromedp.WithPollingInterval(200*time.Millisecond),
		),
	)
	if err != nil {
		g.progressf("⚠️  Timeout waiting for result links: %v (lanjut mencoba...)", err)
	}

	_ = chromedp.Run(scrapeCtx,
		chromedp.WaitVisible("[role=\"main\"]", chromedp.ByQuery),
		chromedp.Sleep(350*time.Millisecond),
	)

	var pageInfo struct {
		URL        string `json:"url"`
		Title      string `json:"title"`
		HasSidebar bool   `json:"hasSidebar"`
		HasResults bool   `json:"hasResults"`
	}
	_ = chromedp.Run(scrapeCtx,
		chromedp.Evaluate(`
			() => ({
				url: window.location.href,
				title: document.title,
				hasSidebar: !!document.querySelector('[role="main"]'),
				hasResults: document.querySelectorAll('div[data-result-index]').length > 0
			})
		`, &pageInfo),
	)
	g.progressf("📄 Page info: url=%s title=%q hasResults=%v", pageInfo.URL, pageInfo.Title, pageInfo.HasResults)

	stores := []types.StoreInfo{}
	scrollAttempts := 0
	maxScrollAttempts := 15

	g.progressLine("🔍 Scrolling dan mengumpulkan data...")

	for len(stores) < maxResults && scrollAttempts < maxScrollAttempts {
		chromedp.Run(scrapeCtx,
			chromedp.Evaluate(`(function(){
				var feed = document.querySelector('div[role="feed"]');
				var s = feed || document.querySelector('[role="main"]') || document.querySelector('.m6QErb');
				if (s) { s.scrollTop = s.scrollHeight; return true; }
				return false;
			})()`, nil),
		)
		time.Sleep(250 * time.Millisecond)

		cardCountInt := g.getCardCount(scrapeCtx)
		if cardCountInt == 0 {
			scrollAttempts++
			time.Sleep(650 * time.Millisecond)
			continue
		}
		g.progressf("🔎 %d kartu", cardCountInt)

		for i := 0; i < cardCountInt && len(stores) < maxResults; i++ {
			cardCtx, cancel := context.WithTimeout(scrapeCtx, 45*time.Second)
			store, err := g.processCard(i, cardCtx)
			cancel()
			if err != nil {
				summary.CardErrors++
				g.progressf("❌ Kartu %d: %v", i, err)
				continue
			}
			if store != nil {
				if store.HasWebsite {
					summary.WithWebsite++
				}
				if strings.TrimSpace(store.Phone) == "" {
					summary.NoPhone++
				}
				stores = append(stores, *store)
				g.reportProgress(len(stores), maxResults)
				g.progressf("✅ [%d] %s - %s", len(stores), store.Name, getPhoneDisplay(store.Phone))
				continue
			}
			summary.SkippedOther++
		}

		scrollAttempts++
		g.progressf("📊 %d/%d (scroll %d/%d)", len(stores), maxResults, scrollAttempts, maxScrollAttempts)
	}

	summary.SavedNoWebsite = len(stores)
	g.logScrapeSummary(summary)
	return stores, summary, nil
}

func (g *GoogleMapsScraper) logScrapeSummary(s ScrapeSummary) {
	g.progressLine("")
	g.progressLine("========== Ringkasan pencarian ==========")
	g.progressf("Yang dicari (kuota maks.):     %d listing", s.TargetMax)
	g.progressf("Yang tersimpan ke file:        %d listing", s.SavedNoWebsite)
	g.progressf("Di antaranya punya website:    %d listing", s.WithWebsite)
	g.progressf("Di antaranya tanpa telepon:    %d listing", s.NoPhone)
	g.progressf("Gagal proses kartu:            %d kali", s.CardErrors)
	g.progressf("Dilewati lainnya *:            %d kali", s.SkippedOther)
	g.progressLine("------------------------------------------")
	g.progressLine("Penjelasan singkat:")
	g.progressf("  • Target: hingga %d listing pertama yang berhasil dibaca dari kartu hasil.", s.TargetMax)
	g.progressf("  • Tersimpan: %d listing.", s.SavedNoWebsite)
	g.progressf("  • Punya website: %d listing.", s.WithWebsite)
	g.progressf("  • Tanpa nomor telepon: %d listing.", s.NoPhone)
	if s.CardErrors > 0 || s.SkippedOther > 0 {
		g.progressf("  • Sisanya: %d error teknis (panel/timeout), %d skip lain (duplikat nama, data kosong, atau klik tidak jalan).", s.CardErrors, s.SkippedOther)
	}
	g.progressLine("  * Angka “dilewati lainnya” tidak membedakan duplikat vs data kosong.")
	g.progressLine("==========================================")
}
