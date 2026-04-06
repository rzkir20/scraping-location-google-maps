package controllers

import (
	"context"
	"sync"
)

// GoogleMapsScraper mengelola sesi chromedp dan deduplikasi hasil.
type GoogleMapsScraper struct {
	ctx            context.Context
	cancel         context.CancelFunc
	lastSearchURL  string
	processedNames map[string]bool
	processedIDs   map[string]bool
	mu             sync.Mutex
	// ProgressLog dipasang dari RunScrapeJob (GUI/CLI) agar pesan proses tampil di log aktivitas.
	ProgressLog func(string)
}

// ScrapeSummary statistik satu sesi ScrapeCoffeeShops.
type ScrapeSummary struct {
	TargetMax      int
	SavedNoWebsite int
	WithWebsite    int
	NoPhone        int
	CardErrors     int
	SkippedOther   int
}
