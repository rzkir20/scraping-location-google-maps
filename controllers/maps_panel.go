package controllers

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// closeDetailPanelClicks mencoba kontrol Maps “kembali ke hasil” (ID/EN).
func (g *GoogleMapsScraper) closeDetailPanelClicks(ctx context.Context) {
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function(){
				var sels = [
					'button[aria-label*="Kembali ke hasil" i]',
					'button[aria-label*="kembali ke hasil" i]',
					'button[aria-label*="Back to results" i]',
					'button[aria-label="Kembali" i]',
					'button[aria-label*="Kembali" i]',
					'button[aria-label*="Back" i]',
					'button[jsaction*="back"]'
				];
				var clicked = 0;
				for (var round = 0; round < 2; round++) {
					for (var i = 0; i < sels.length; i++) {
						var b = document.querySelector(sels[i]);
						if (b && b.offsetParent !== null) {
							b.click();
							clicked++;
							break;
						}
					}
				}
				if (clicked === 0) {
					for (var k = 0; k < 3; k++) {
						document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape', keyCode: 27, bubbles: true }));
					}
				}
				return true;
			})()
		`, nil),
	)
	time.Sleep(450 * time.Millisecond)
}

func feedReadyPoll() chromedp.Action {
	return chromedp.Poll(
		`(function(){
			var f = document.querySelector('div[role="feed"]');
			if (!f) return false;
			return f.querySelectorAll('div[role="article"] a[href*="/maps/place/"]').length > 0;
		})()`,
		nil,
		chromedp.WithPollingTimeout(18*time.Second),
		chromedp.WithPollingInterval(120*time.Millisecond),
	)
}

// restoreListAfterDetail menutup panel tempat dan menunggu feed bisa dipakai lagi.
func (g *GoogleMapsScraper) restoreListAfterDetail(ctx context.Context) {
	g.closeDetailPanelClicks(ctx)
	if err := chromedp.Run(ctx, feedReadyPoll()); err == nil {
		time.Sleep(200 * time.Millisecond)
		return
	}
	// NavigateBack sering keluar dari halaman hasil → kartu berikutnya gagal massal; pakai ulang URL pencarian.
	if g.lastSearchURL != "" {
		log.Println("⚠️  Daftar hasil belum muncul; muat ulang halaman pencarian...")
		_ = chromedp.Run(ctx,
			chromedp.Navigate(g.lastSearchURL),
			chromedp.WaitVisible("body", chromedp.ByQuery),
			chromedp.Sleep(2*time.Second),
		)
		g.dismissBlockingUI(ctx)
		_ = chromedp.Run(ctx,
			chromedp.Poll(
				`(function(){
					var feed = document.querySelector('div[role="feed"]');
					if (feed && feed.querySelector('a[href*="/maps/place/"]')) return true;
					if (document.querySelectorAll('a[href*="/maps/place/"]').length > 0) return true;
					return false;
				})()`,
				nil,
				chromedp.WithPollingTimeout(40*time.Second),
				chromedp.WithPollingInterval(300*time.Millisecond),
			),
		)
		_ = chromedp.Run(ctx, feedReadyPoll())
		time.Sleep(250 * time.Millisecond)
		return
	}
	log.Println("⚠️  Daftar hasil belum muncul (tidak ada URL pencarian tersimpan).")
	time.Sleep(500 * time.Millisecond)
}
