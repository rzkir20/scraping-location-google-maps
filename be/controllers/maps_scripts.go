package controllers

// jsPlaceCards mengembalikan link /maps/place/ unik (feed dulu). Harus selaras urutan klik getCardCount / processCard.
const jsPlaceCardsFn = `
function __gmapsPlaceCards() {
	var feed = document.querySelector('div[role="feed"]');
	var links = [];
	var seen = {};
	function isJunkListLink(a) {
		var t = (a.textContent || '').replace(/\s+/g, ' ').trim();
		var lab = (a.getAttribute('aria-label') || '').replace(/\s+/g, ' ').trim();
		var low = (t + ' | ' + lab).toLowerCase();
		if (/^results$|^search results$|^hasil$|^hasil penelusuran$|^tempat$|^places$|^more results$/i.test(t)) return true;
		if (/^results$|^search results$|^hasil penelusuran$/i.test(lab)) return true;
		if (lab.indexOf('results') === 0 && lab.length < 24) return true;
		if (low.indexOf('search results') >= 0 && t.length < 30) return true;
		if (!lab && !t) return true;
		return false;
	}
	function addLink(a) {
		if (!a || isJunkListLink(a)) return;
		var href = (a.getAttribute('href') || '').trim();
		if (href.indexOf('/maps/place/') < 0) return;
		var key = href.split('?')[0].split('#')[0];
		if (!key || seen[key]) return;
		seen[key] = true;
		links.push(a);
	}
	var i;
	if (feed) {
		var articles = feed.querySelectorAll('div[role="article"]');
		if (articles.length > 0) {
			for (i = 0; i < articles.length; i++) {
				var art = articles[i];
				var placeA = art.querySelector('a.hfpxzc[href*="/maps/place/"]') ||
					art.querySelector('a[href*="/maps/place/"]');
				if (placeA) addLink(placeA);
			}
		} else {
			var nodes = feed.querySelectorAll('a[href*="/maps/place/"]');
			for (i = 0; i < nodes.length; i++) addLink(nodes[i]);
		}
	}
	if (links.length === 0) {
		var all = document.querySelectorAll('a[href*="/maps/place/"]');
		for (i = 0; i < all.length; i++) addLink(all[i]);
	}
	return links;
}
`
