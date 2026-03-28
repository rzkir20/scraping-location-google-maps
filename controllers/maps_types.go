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
}

// ScrapeSummary statistik satu sesi ScrapeCoffeeShops.
type ScrapeSummary struct {
	TargetMax      int
	SavedNoWebsite int
	WithWebsite    int
	CardErrors     int
	SkippedOther   int
}
