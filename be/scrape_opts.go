package main

// ScrapeJobOptions mengatur perilaku browser dan callback progres (dipakai dari RunScrapeJob).
type ScrapeJobOptions struct {
	// Headless: true = Chrome tanpa jendela (disarankan untuk API + FE).
	Headless bool
	// OnProgress: savedCount = listing yang lolos filter; targetMax = kuota job.
	OnProgress func(savedCount, targetMax int)
}
