package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"location/types"
)

// scrapeRunner menyimpan satu job aktif (satu scraping pada satu waktu).
var scrapeRunner struct {
	mu         sync.Mutex
	jobID      string
	status     string // idle | running | done | error
	logs       []string
	stores     []types.StoreInfo
	errMsg     string
	keyword    string
	location   string
	savedCount int
	targetMax  int
	mapLat     string
	mapLng     string
}

func newJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func appendScrapeLog(line string) {
	scrapeRunner.mu.Lock()
	defer scrapeRunner.mu.Unlock()
	scrapeRunner.logs = append(scrapeRunner.logs, line)
}

// handleAPIScrape: POST /api/scrape — mulai job (202 + jobId). 409 jika masih running.
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

	scrapeRunner.mu.Lock()
	if scrapeRunner.status == "running" {
		scrapeRunner.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "scraping masih berjalan, tunggu selesai"})
		return
	}
	jobID := newJobID()
	scrapeRunner.jobID = jobID
	scrapeRunner.status = "running"
	scrapeRunner.logs = nil
	scrapeRunner.stores = nil
	scrapeRunner.errMsg = ""
	scrapeRunner.keyword = keyword
	scrapeRunner.location = location
	scrapeRunner.savedCount = 0
	scrapeRunner.targetMax = maxResults
	if geoErr == nil {
		scrapeRunner.mapLat, scrapeRunner.mapLng = latStr, lngStr
	} else {
		scrapeRunner.mapLat, scrapeRunner.mapLng = "", ""
	}
	scrapeRunner.mu.Unlock()

	go runScrapeJobHTTP(keyword, location, maxResults)

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

func runScrapeJobHTTP(keyword, location string, maxResults int) {
	logf := func(s string) { appendScrapeLog(s) }
	onProgress := func(saved, target int) {
		scrapeRunner.mu.Lock()
		scrapeRunner.savedCount = saved
		if target > 0 {
			scrapeRunner.targetMax = target
		}
		scrapeRunner.mu.Unlock()
	}
	stores, err := RunScrapeJob(keyword, location, maxResults, logf, false, ScrapeJobOptions{
		Headless:   true,
		OnProgress: onProgress,
	})

	scrapeRunner.mu.Lock()
	defer scrapeRunner.mu.Unlock()
	if err != nil {
		scrapeRunner.status = "error"
		scrapeRunner.errMsg = err.Error()
		scrapeRunner.stores = nil
		return
	}
	scrapeRunner.status = "done"
	scrapeRunner.stores = stores
	scrapeRunner.savedCount = len(stores)
	scrapeRunner.targetMax = maxResults
	scrapeRunner.errMsg = ""
}

// handleScrapeStatus: GET /api/scrape/status?jobId=...
func handleScrapeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	scrapeRunner.mu.Lock()
	if jobID == "" || jobID != scrapeRunner.jobID {
		scrapeRunner.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job tidak ditemukan"})
		return
	}
	logs := append([]string(nil), scrapeRunner.logs...)
	status := scrapeRunner.status
	storesCopy := append([]types.StoreInfo(nil), scrapeRunner.stores...)
	errMsg := scrapeRunner.errMsg
	kw := scrapeRunner.keyword
	loc := scrapeRunner.location
	saved := scrapeRunner.savedCount
	target := scrapeRunner.targetMax
	mapLat := scrapeRunner.mapLat
	mapLng := scrapeRunner.mapLng
	scrapeRunner.mu.Unlock()

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
