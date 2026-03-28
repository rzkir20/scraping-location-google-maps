package main

import (
	"fmt"
	"log"

	"location/controllers"
	"location/types"
)

// runScrapeJob menjalankan geocoding, scrape, dan simpan hasil.
// Jika logStores false, daftar per-toko tidak ditulis lewat logf (mis. mode GUI pakai tabel).
func runScrapeJob(keyword, locationName string, maxResults int, logf func(string), logStores bool) ([]types.StoreInfo, error) {
	if logf == nil {
		logf = func(s string) { log.Print(s) }
	}
	if keyword == "" {
		return nil, fmt.Errorf("keyword tidak boleh kosong")
	}
	if locationName == "" {
		return nil, fmt.Errorf("nama lokasi wajib diisi")
	}
	if maxResults < 1 {
		return nil, fmt.Errorf("target minimal 1")
	}

	logf(fmt.Sprintf("Mencari koordinat untuk: %s ...", locationName))
	lat, lng, err := geocodeLocation(locationName)
	if err != nil {
		return nil, fmt.Errorf("geocoding gagal: %w", err)
	}
	logf(fmt.Sprintf("Koordinat: %s, %s", lat, lng))

	searchURL := buildSearchURL(keyword, lat, lng)
	logf(fmt.Sprintf("Keyword: %s", keyword))
	logf(fmt.Sprintf("Target listing (tanpa website): %d", maxResults))
	logf(fmt.Sprintf("URL: %s", searchURL))

	scraper, err := controllers.NewGoogleMapsScraper()
	if err != nil {
		return nil, fmt.Errorf("browser: %w", err)
	}
	defer scraper.Close()

	if err := scraper.Init(); err != nil {
		return nil, fmt.Errorf("inisialisasi: %w", err)
	}

	stores, summary, err := scraper.ScrapeCoffeeShops(searchURL, maxResults)
	if err != nil {
		return nil, fmt.Errorf("scraping: %w", err)
	}

	logf(fmt.Sprintf("Total tersimpan: %d dari target maks. %d · dilewati (ada website): %d",
		len(stores), summary.TargetMax, summary.WithWebsite))
	if logStores {
		logf("Hasil:")
		for i, store := range stores {
			logf(fmt.Sprintf("%d. %s - Phone: %s", i+1, store.Name, getPhoneDisplay(store.Phone)))
		}
	} else if len(stores) > 0 {
		logf(fmt.Sprintf("%d listing — lihat tabel di bawah.", len(stores)))
	}

	if err := scraper.SaveToFile(stores, "results.json"); err != nil {
		logf(fmt.Sprintf("Error simpan JSON: %v", err))
	} else {
		logf("Disimpan: results.json")
	}
	if err := scraper.SaveToCSV(stores, "results.csv"); err != nil {
		logf(fmt.Sprintf("⚠ Error simpan CSV: %v", err))
	} else {
		logf("Disimpan: results.csv (hanya baris dengan nomor)")
	}

	return stores, nil
}
