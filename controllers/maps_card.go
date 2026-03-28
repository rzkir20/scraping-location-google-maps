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
			`(function(){
				var h = document.querySelector('h1.DUwDvf') ||
					document.querySelector('[role="main"] h1.DUwDvf') ||
					document.querySelector('h1[class*="DUwDvf"]');
				if (!h) return false;
				var tx = (h.textContent || '').trim();
				if (!tx) return false;
				if (/^bersponsor$/i.test(tx) || /^results$/i.test(tx) || /^hasil$/i.test(tx)) return false;
				return true;
			})()`,
			nil,
			chromedp.WithPollingTimeout(9*time.Second),
			chromedp.WithPollingInterval(120*time.Millisecond),
		),
	)
	cancel()
	if err != nil {
		g.restoreListAfterDetail(ctx)
		return nil, fmt.Errorf("panel tidak muncul")
	}
	time.Sleep(400 * time.Millisecond)

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
			return JSON.stringify({ name: name, phone: phone, hasWebsite: hasWebsite, address: address });
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
		Address:    strings.TrimSpace(data.Address),
		HasWebsite: data.HasWebsite,
	}, nil
}
