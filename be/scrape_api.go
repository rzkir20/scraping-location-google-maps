package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"location/types"
)

// scrapeJobState: satu entri per jobId (boleh banyak job paralel).
type scrapeJobState struct {
	status      string // idle | running | done | error
	logs        []string
	stores      []types.StoreInfo
	errMsg      string
	keyword     string
	location    string
	savedCount  int
	targetMax   int
	mapLat      string
	mapLng      string
	currentCard types.LiveCard
	finishedAt  time.Time
}

var (
	jobsMu sync.Mutex
	jobs   = make(map[string]*scrapeJobState)

	// scrapeSlots membatasi jumlah Chrome paralel (0 = tanpa batas, dari env MAX_CONCURRENT_SCRAPES).
	scrapeSlots chan struct{}
)

func init() {
	n := maxConcurrentScrapesFromEnv()
	if n > 0 {
		scrapeSlots = make(chan struct{}, n)
	}
}

func maxConcurrentScrapesFromEnv() int {
	s := strings.TrimSpace(os.Getenv("MAX_CONCURRENT_SCRAPES"))
	if s == "" {
		return 12
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 12
	}
	return n
}

func tryAcquireScrapeSlot() bool {
	if scrapeSlots == nil {
		return true
	}
	select {
	case scrapeSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseScrapeSlot() {
	if scrapeSlots == nil {
		return
	}
	<-scrapeSlots
}

func newJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// pruneFinishedJobs menghapus job selesai/error yang sudah lama (hindari map membengkak).
func pruneFinishedJobs() {
	const maxAge = 45 * time.Minute
	now := time.Now()
	for id, j := range jobs {
		if j == nil {
			delete(jobs, id)
			continue
		}
		if j.status != "running" && !j.finishedAt.IsZero() && now.Sub(j.finishedAt) > maxAge {
			delete(jobs, id)
		}
	}
}

func appendJobLog(jobID, line string) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	if j := jobs[jobID]; j != nil {
		j.logs = append(j.logs, line)
	}
}

// handleAPIScrape: POST /api/scrape — mulai job (202 + jobId). Banyak job boleh paralel.
func handleAPIScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	const maxBody = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad body"})
		return
	}
	var body struct {
		Keyword    string `json:"keyword"`
		Location   string `json:"location"`
		MaxResults int    `json:"maxResults"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	keyword := strings.TrimSpace(body.Keyword)
	location := strings.TrimSpace(body.Location)
	maxResults := body.MaxResults
	if maxResults < 1 {
		maxResults = defaultMaxResults
	}

	if keyword == "" || location == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "keyword dan location wajib diisi"})
		return
	}

	latStr, lngStr, geoErr := geocodeLocation(location)

	if !tryAcquireScrapeSlot() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "server penuh: terlalu banyak scraping paralel. Coba lagi sebentar atau naikkan MAX_CONCURRENT_SCRAPES.",
		})
		return
	}

	jobID := newJobID()

	jobsMu.Lock()
	pruneFinishedJobs()
	jobs[jobID] = &scrapeJobState{
		status:     "running",
		keyword:    keyword,
		location:   location,
		savedCount: 0,
		targetMax:  maxResults,
	}
	if geoErr == nil {
		jobs[jobID].mapLat, jobs[jobID].mapLng = latStr, lngStr
	}
	jobsMu.Unlock()

	go runScrapeJobHTTP(jobID, keyword, location, maxResults)

	resp := map[string]any{
		"jobId":   jobID,
		"status":  "running",
		"message": "Gunakan GET /api/scrape/status?jobId=... untuk memantau",
	}
	if geoErr == nil {
		resp["mapCenter"] = map[string]string{"lat": latStr, "lng": lngStr}
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func runScrapeJobHTTP(jobID, keyword, location string, maxResults int) {
	defer releaseScrapeSlot()

	logf := func(s string) { appendJobLog(jobID, s) }
	onProgress := func(saved, target int) {
		jobsMu.Lock()
		if j := jobs[jobID]; j != nil {
			j.savedCount = saved
			if target > 0 {
				j.targetMax = target
			}
		}
		jobsMu.Unlock()
	}
	onCurrentCard := func(card types.LiveCard) {
		jobsMu.Lock()
		if j := jobs[jobID]; j != nil {
			j.currentCard = card
		}
		jobsMu.Unlock()
	}
	stores, err := RunScrapeJob(keyword, location, maxResults, logf, false, ScrapeJobOptions{
		Headless:         true,
		ResultFileSuffix: jobID,
		OnProgress:       onProgress,
		OnCurrentCard:    onCurrentCard,
	})

	jobsMu.Lock()
	defer jobsMu.Unlock()
	j := jobs[jobID]
	if j == nil {
		return
	}
	now := time.Now()
	j.finishedAt = now
	if err != nil {
		j.status = "error"
		j.errMsg = err.Error()
		j.stores = nil
		return
	}
	j.status = "done"
	j.stores = stores
	j.savedCount = len(stores)
	j.targetMax = maxResults
	j.errMsg = ""
}

// handleScrapeStatus: GET /api/scrape/status?jobId=...
func handleScrapeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	jobsMu.Lock()
	j := jobs[jobID]
	if jobID == "" || j == nil {
		jobsMu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job tidak ditemukan"})
		return
	}
	logs := append([]string(nil), j.logs...)
	status := j.status
	storesCopy := append([]types.StoreInfo(nil), j.stores...)
	errMsg := j.errMsg
	kw := j.keyword
	loc := j.location
	saved := j.savedCount
	target := j.targetMax
	mapLat := j.mapLat
	mapLng := j.mapLng
	currentCard := j.currentCard
	jobsMu.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := map[string]any{
		"jobId":    jobID,
		"status":   status,
		"logs":     logs,
		"keyword":  kw,
		"location": loc,
		"progress": map[string]int{"saved": saved, "target": target},
	}
	if mapLat != "" && mapLng != "" {
		resp["mapCenter"] = map[string]string{"lat": mapLat, "lng": mapLng}
	}
	if currentCard.Name != "" {
		resp["currentCard"] = currentCard
	}
	if status == "done" {
		resp["stores"] = storesCopy
	}
	if status == "error" {
		resp["error"] = errMsg
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
