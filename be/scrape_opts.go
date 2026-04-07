package main

import "location/types"

// ScrapeJobOptions mengatur perilaku browser dan callback progres (dipakai dari RunScrapeJob).
type ScrapeJobOptions struct {
	// Headless: true = Chrome tanpa jendela (disarankan untuk API + FE).
	Headless bool
	// ResultFileSuffix — jika non-empty, simpan ke results_<suffix>.json / .csv (hindari bentrok job paralel).
	ResultFileSuffix string
	// OnProgress: savedCount = listing yang lolos filter; targetMax = kuota job.
	OnProgress func(savedCount, targetMax int)
	// OnCurrentCard: data kartu yang sedang diproses untuk live tracking.
	OnCurrentCard func(card types.LiveCard)
}
