package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"location/types"

	"github.com/chromedp/chromedp"
)

// placeKeyAt mengembalikan kunci unik /maps/place/... untuk kartu di indeks (kosong jika tidak ada).
func (g *GoogleMapsScraper) placeKeyAt(ctx context.Context, index int) string {
	var raw []byte
	js := fmt.Sprintf(jsPlaceCardsFn+`
		(function(){
			var cards = __gmapsPlaceCards();
			var card = cards[%d];
			if (!card) return '';
			var href = (card.getAttribute('href') || '').trim();
			return href.split('?')[0].split('#')[0];
		})()
	`, index)
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &raw)); err != nil {
		return ""
	}
	var key string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &key)
	}
	return strings.TrimSpace(key)
}

func (g *GoogleMapsScraper) getCardCount(ctx context.Context) int {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(jsPlaceCardsFn+`
		(function(){ return __gmapsPlaceCards().length; })()
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
	placeKey := g.placeKeyAt(ctx, index)
	if placeKey != "" {
		g.mu.Lock()
		skip := g.processedIDs[placeKey]
		g.mu.Unlock()
		if skip {
			return nil, nil
		}
	}

	g.mu.Lock()
	prevCardName := strings.TrimSpace(g.lastCardName)
	g.mu.Unlock()

	clickJS := fmt.Sprintf(jsPlaceCardsFn+`
		(function(){
			var cards = __gmapsPlaceCards();
			var card = cards[%d];
			if (!card) return false;
			card.scrollIntoView({ block: 'center', inline: 'nearest' });
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

	panelCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := chromedp.Run(panelCtx,
		chromedp.Poll(
			fmt.Sprintf(`(function(){
				var h = document.querySelector('h1.DUwDvf') ||
					document.querySelector('[role="main"] h1.DUwDvf') ||
					document.querySelector('h1[class*="DUwDvf"]');
				if (!h) return false;
				var tx = (h.textContent || '').trim();
				if (!tx) return false;
				if (/^bersponsor$/i.test(tx) || /^results$/i.test(tx) || /^hasil$/i.test(tx)) return false;
				var prev = %q;
				if (prev && tx.toLowerCase() === String(prev).toLowerCase()) return false;
				return true;
			})()`, prevCardName),
			nil,
			chromedp.WithPollingTimeout(8*time.Second),
			chromedp.WithPollingInterval(90*time.Millisecond),
		),
	)
	cancel()
	if err != nil {
		g.restoreListAfterDetail(ctx)
		return nil, fmt.Errorf("panel tidak muncul")
	}
	time.Sleep(220 * time.Millisecond)

	getDataJS := `
		(function(){
			var name = '';
			var el = null;
			var cands = ['h1.DUwDvf','[role="main"] h1.DUwDvf','h1[data-value="name"]'];
			for (var ci = 0; ci < cands.length; ci++) {
				el = document.querySelector(cands[ci]);
				if (el && (el.textContent||'').trim()) break;
				el = null;
			}
			if (!el) {
				var hs = document.querySelectorAll('div[role="main"] h1');
				for (var hi = 0; hi < hs.length; hi++) {
					var tx = (hs[hi].textContent || '').trim();
					if (!tx) continue;
					if (/^bersponsor$/i.test(tx)) continue;
					if (/^results$/i.test(tx) || /^hasil( penelusuran)?$/i.test(tx) || /^search results$/i.test(tx)) continue;
					el = hs[hi];
					break;
				}
			}
			if (!el) el = document.querySelector('h1');
			if (el) name = (el.textContent || '').trim();
			var phone = '';
			var tel = document.querySelector('a[href^="tel:"]');
			if (tel) {
				phone = (tel.getAttribute('href') || '').replace(/^tel:\\s*/i,'').trim();
			} else {
				var btn = document.querySelector('button[data-item-id*="phone"]');
				if (btn) {
					var did = (btn.getAttribute('data-item-id') || '');
					var m = did.match(/tel:([0-9+]+)/i);
					if (m) phone = m[1];
					else {
						var row = btn.querySelector('.Io6YTe');
						if (row) phone = (row.textContent || '').trim();
						if (!phone) phone = (btn.getAttribute('aria-label') || btn.textContent || '').replace(/^[^:]*:\\s*/i,'').trim();
						phone = phone.replace(/[^0-9+\\-\\s()]/g,'').trim();
					}
				}
			}
			var hasWebsite = false;
			var auth = document.querySelector('a[data-item-id="authority"]');
			if (auth) {
				var ah = (auth.getAttribute('href') || '').toLowerCase();
				if (ah.indexOf('google.com') < 0 && ah.indexOf('g.page') < 0 && ah.indexOf('maps.google') < 0)
					hasWebsite = true;
			}
			if (!hasWebsite) {
				var links = document.querySelectorAll('[role="main"] a[href^="http"]');
				for (var i = 0; i < links.length; i++) {
					var h = (links[i].getAttribute('href') || '').toLowerCase();
					if (h.indexOf('google.') >= 0 || h.indexOf('gstatic.') >= 0 || h.indexOf('schema.org') >= 0) continue;
					if (h.indexOf('http') === 0 && h.length > 10) { hasWebsite = true; break; }
				}
			}
			var address = '';
			var addrBtn = document.querySelector('button[data-item-id="address"]') ||
				document.querySelector('button[data-item-id*="address"]');
			if (addrBtn) {
				var row = addrBtn.querySelector('.Io6YTe');
				if (row) address = (row.textContent || '').trim();
				if (!address) {
					var al = (addrBtn.getAttribute('aria-label') || '').trim();
					if (al) address = al.replace(/^[^:]+:\s*/i, '').trim();
				}
			}
			var category = '';
			var catSel = [
				'button[jsaction*="pane.rating.category"]',
				'button[jsaction*="pane.placeActions.category"]',
				'[role="main"] button[aria-label*="Kategori" i]',
				'[role="main"] button[aria-label*="Category" i]'
			];
			for (var ci2 = 0; ci2 < catSel.length; ci2++) {
				var catEl = document.querySelector(catSel[ci2]);
				if (!catEl) continue;
				var catTx = (catEl.textContent || '').trim();
				if (catTx) { category = catTx; break; }
			}
			if (!category) {
				var catFallback = document.querySelector('[role="main"] .DkEaL') || document.querySelector('[role="main"] .RWPxGd');
				if (catFallback) category = (catFallback.textContent || '').trim();
			}
			var openingStatus = '';
			var openNodes = document.querySelectorAll('[role="main"] span, [role="main"] div');
			for (var oi = 0; oi < openNodes.length; oi++) {
				var ot = (openNodes[oi].textContent || '').trim();
				if (!ot) continue;
				if (/^(buka|tutup|open|closed)\\b/i.test(ot)) {
					openingStatus = ot;
					break;
				}
			}
			function __ratingNum(s) {
				if (!s) return '';
				var m = s.match(/([0-9]+(?:[.,][0-9]+)?)/);
				if (!m) return '';
				var v = parseFloat(String(m[1]).replace(',', '.'));
				if (v >= 1 && v <= 5) return String(m[1]).replace(',', '.');
				return '';
			}
			function __ratingFromLabel(s) {
				if (!s) return '';
				var low = s.toLowerCase();
				if (low.indexOf('star') >= 0 || low.indexOf('bintang') >= 0 || low.indexOf('rating') >= 0 ||
					low.indexOf('penilaian') >= 0 || low.indexOf('ulasan') >= 0 || low.indexOf('review') >= 0 ||
					/\bout of\b/i.test(s) || /^[0-9]+[.,][0-9]+\s*\/\s*5/.test(low)) {
					return __ratingNum(s);
				}
				return '';
			}
			var rating = '';
			var ratingBtnSels = [
				'button[jsaction*="pane.rating.moreReviews"]',
				'button[jsaction*="pane.rating.place"]',
				'button[jsaction*="pane.rating"]',
				'[role="main"] button[jsaction*="pane.rating"]',
				'[role="main"] a[jsaction*="pane.rating"]'
			];
			for (var rbi = 0; rbi < ratingBtnSels.length && !rating; rbi++) {
				var rb = document.querySelector(ratingBtnSels[rbi]);
				if (!rb) continue;
				var ral = (rb.getAttribute('aria-label') || '').trim();
				rating = __ratingFromLabel(ral);
				if (!rating) rating = __ratingNum(ral);
			}
			if (!rating) {
				var rImgs = document.querySelectorAll('[role="main"] [role="img"][aria-label]');
				for (var rii = 0; rii < rImgs.length && !rating; rii++) {
					var ial = (rImgs[rii].getAttribute('aria-label') || '').trim();
					rating = __ratingFromLabel(ial);
				}
			}
			if (!rating) {
				var h1r = document.querySelector('[role="main"] h1.DUwDvf') || document.querySelector('[role="main"] h1');
				var sec = h1r ? h1r.parentElement : null;
				for (var d = 0; d < 5 && sec && !rating; d++) {
					var ns = sec.querySelectorAll('span, [role="img"]');
					for (var nj = 0; nj < ns.length && nj < 40 && !rating; nj++) {
						var eln = ns[nj];
						var al2 = (eln.getAttribute('aria-label') || '').trim();
						if (al2) rating = __ratingFromLabel(al2);
						if (!rating) {
							var tx2 = (eln.textContent || '').trim();
							if (tx2.length >= 3 && tx2.length <= 5 && /^[0-9][.,][0-9]$/.test(tx2)) {
								var v2 = parseFloat(tx2.replace(',', '.'));
								if (v2 >= 1 && v2 <= 5) rating = tx2.replace(',', '.');
							}
						}
					}
					sec = sec.parentElement;
				}
			}
			return JSON.stringify({ name: name, phone: phone, hasWebsite: hasWebsite, address: address, category: category, openingStatus: openingStatus, rating: rating });
		})()
	`
	var raw []byte
	if err := chromedp.Run(ctx, chromedp.Evaluate(getDataJS, &raw)); err != nil {
		g.restoreListAfterDetail(ctx)
		return nil, err
	}
	var data struct {
		Name       string `json:"name"`
		Phone      string `json:"phone"`
		Address    string `json:"address"`
		Rating     string `json:"rating"`
		Category   string `json:"category"`
		Open       string `json:"openingStatus"`
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

	g.restoreListAfterDetail(ctx)

	name := strings.TrimSpace(data.Name)
	if name == "" || isJunkPlaceTitle(name) {
		if placeKey != "" {
			g.mu.Lock()
			g.processedIDs[placeKey] = true
			g.mu.Unlock()
		}
		return nil, nil
	}
	g.mu.Lock()
	g.lastCardName = name
	g.mu.Unlock()
	g.reportCurrentCard(types.LiveCard{
		Name:          name,
		Rating:        strings.TrimSpace(data.Rating),
		Category:      strings.TrimSpace(data.Category),
		Address:       strings.TrimSpace(data.Address),
		Phone:         strings.TrimSpace(data.Phone),
		OpeningStatus: strings.TrimSpace(data.Open),
	})
	g.mu.Lock()
	if g.processedNames[name] {
		g.mu.Unlock()
		if placeKey != "" {
			g.mu.Lock()
			g.processedIDs[placeKey] = true
			g.mu.Unlock()
		}
		return nil, nil
	}
	g.processedNames[name] = true
	if placeKey != "" {
		g.processedIDs[placeKey] = true
	}
	g.mu.Unlock()

	return &types.StoreInfo{
		Name:       name,
		Rating:     strings.TrimSpace(data.Rating),
		Phone:      strings.TrimSpace(data.Phone),
		Address:    strings.TrimSpace(data.Address),
		HasWebsite: data.HasWebsite,
	}, nil
}
