package main

import (
	"fmt"
	"log"

	"location/controllers"
	"location/types"
)

// RunScrapeJob menjalankan geocoding, scrape, dan simpan hasil.
// Jika logStores false, daftar per-toko tidak ditulis lewat logf (mis. mode GUI pakai tabel).
func RunScrapeJob(keyword, locationName string, maxResults int, logf func(string), logStores bool, opts ScrapeJobOptions) ([]types.StoreInfo, error) {
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
	logf(fmt.Sprintf("Target listing (tanpa website, wajib ada nomor): %d", maxResults))
	logf(fmt.Sprintf("URL: %s", searchURL))

	scraper, err := controllers.NewGoogleMapsScraper(opts.Headless)
	if err != nil {
		return nil, fmt.Errorf("browser: %w", err)
	}
	scraper.ProgressLog = logf
	scraper.OnProgress = opts.OnProgress
	scraper.OnCurrentCard = opts.OnCurrentCard
	defer scraper.Close()

	if err := scraper.Init(); err != nil {
		return nil, fmt.Errorf("inisialisasi: %w", err)
	}

	stores, summary, err := scraper.ScrapeCoffeeShops(searchURL, maxResults)
	if err != nil {
		return nil, fmt.Errorf("scraping: %w", err)
	}

	logf(fmt.Sprintf("Total tersimpan: %d dari target maks. %d · dilewati (website): %d · dilewati (tanpa telepon): %d",
		len(stores), summary.TargetMax, summary.WithWebsite, summary.NoPhone))
	if logStores {
		logf("Hasil:")
		for i, store := range stores {
			addr := store.Address
			if addr == "" {
				addr = "N/A"
			}
			logf(fmt.Sprintf("%d. %s - Phone: %s - Alamat: %s", i+1, store.Name, getPhoneDisplay(store.Phone), addr))
		}
	} else if len(stores) > 0 {
		logf(fmt.Sprintf("%d listing — lihat tabel di bawah.", len(stores)))
	}

	if err := scraper.SaveToFile(stores, "results.json"); err != nil {
		logf(fmt.Sprintf("Error simpan JSON: %v", err))
	}
	if err := scraper.SaveToCSV(stores, "results.csv"); err != nil {
		logf(fmt.Sprintf("⚠ Error simpan CSV: %v", err))
	}

	return stores, nil
}
