package controllers

import (
	"context"
	"fmt"
	"log"
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
	time.Sleep(600 * time.Millisecond)
}

func (g *GoogleMapsScraper) ScrapeCoffeeShops(url string, maxResults int) ([]types.StoreInfo, ScrapeSummary, error) {
	summary := ScrapeSummary{TargetMax: maxResults}
	log.Printf("📍 Navigating to: %s\n", url)
	log.Printf("📋 Filter: hanya menyimpan yang belum punya website (kuota maks. %d listing)\n", maxResults)

	// Banyak kartu × buka panel butuh waktu; sesuaikan dengan target (tanpa plafon artifisial).
	scrapeBudget := 10*time.Minute + time.Duration(maxResults)*22*time.Second
	scrapeCtx, cancel := context.WithTimeout(g.ctx, scrapeBudget)
	defer cancel()

	err := chromedp.Run(scrapeCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		return nil, summary, fmt.Errorf("failed to navigate: %w", err)
	}
	g.lastSearchURL = url

	g.dismissBlockingUI(scrapeCtx)

	log.Println("⏳ Waiting for search results to load...")
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
			chromedp.WithPollingInterval(300*time.Millisecond),
		),
	)
	if err != nil {
		log.Printf("⚠️  Timeout waiting for result links: %v (lanjut mencoba...)\n", err)
	}

	_ = chromedp.Run(scrapeCtx,
		chromedp.WaitVisible("[role=\"main\"]", chromedp.ByQuery),
	)
	_ = chromedp.Run(scrapeCtx,
		chromedp.Sleep(1*time.Second),
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
	log.Printf("📄 Page info: url=%s title=%q hasResults=%v\n", pageInfo.URL, pageInfo.Title, pageInfo.HasResults)

	stores := []types.StoreInfo{}
	scrollAttempts := 0
	maxScrollAttempts := 15

	log.Println("🔍 Scrolling dan mengumpulkan data...")

	for len(stores) < maxResults && scrollAttempts < maxScrollAttempts {
		chromedp.Run(scrapeCtx,
			chromedp.Evaluate(`(function(){
				var feed = document.querySelector('div[role="feed"]');
				var s = feed || document.querySelector('[role="main"]') || document.querySelector('.m6QErb');
				if (s) { s.scrollTop = s.scrollHeight; return true; }
				return false;
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
			cardCtx, cancel := context.WithTimeout(scrapeCtx, 45*time.Second)
			store, err := g.processCard(i, cardCtx)
			cancel()
			if err != nil {
				summary.CardErrors++
				log.Printf("❌ Kartu %d: %v\n", i, err)
				continue
			}
			if store != nil {
				if store.HasWebsite {
					summary.WithWebsite++
					log.Printf("⏭️  Lewati %s (punya website, tidak dimasukkan)\n", store.Name)
					continue
				}
				stores = append(stores, *store)
				log.Printf("✅ [%d] %s - %s\n", len(stores), store.Name, getPhoneDisplay(store.Phone))
				continue
			}
			summary.SkippedOther++
		}

		scrollAttempts++
		log.Printf("📊 %d/%d (scroll %d/%d)\n", len(stores), maxResults, scrollAttempts, maxScrollAttempts)
	}

	summary.SavedNoWebsite = len(stores)
	g.logScrapeSummary(summary)
	return stores, summary, nil
}

func (g *GoogleMapsScraper) logScrapeSummary(s ScrapeSummary) {
	log.Println("")
	log.Println("========== Ringkasan pencarian ==========")
	log.Printf("Yang dicari (kuota maks.):     %d listing tanpa website\n", s.TargetMax)
	log.Printf("Yang tersimpan ke file:        %d listing\n", s.SavedNoWebsite)
	log.Printf("Dilewati (punya website):      %d listing\n", s.WithWebsite)
	log.Printf("Gagal proses kartu:            %d kali\n", s.CardErrors)
	log.Printf("Dilewati lainnya *:            %d kali\n", s.SkippedOther)
	log.Println("------------------------------------------")
	log.Println("Penjelasan singkat:")
	log.Printf("  • Target: mengumpulkan hingga %d usaha yang di Maps terlihat belum/tidak punya situs web sendiri (menurut panel detail).\n", s.TargetMax)
	log.Printf("  • Tersimpan: %d listing (lolos filter tanpa website).\n", s.SavedNoWebsite)
	log.Printf("  • Punya website (dilewati): %d listing — tidak dimasukkan CSV/JSON sesuai aturan filter.\n", s.WithWebsite)
	if s.CardErrors > 0 || s.SkippedOther > 0 {
		log.Printf("  • Sisanya: %d error teknis (panel/timeout), %d skip lain (duplikat nama, data kosong, atau klik tidak jalan).\n", s.CardErrors, s.SkippedOther)
	}
	log.Println("  * Angka “dilewati lainnya” tidak membedakan duplikat vs data kosong.")
	log.Println("==========================================")
}
