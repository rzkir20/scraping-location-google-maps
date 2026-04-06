package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

const defaultHTTPAddr = "127.0.0.1:8080"

// runHTTPServer mendengarkan di addr (mis. "127.0.0.1:8080") untuk dipanggil dari FE lewat fetch.
// CORS dev: semua origin http://localhost:* , http://127.0.0.1:* , http://[::1]:* (port bebas, cocok Astro 4321/5173/dll).
// Produksi: set SCRAPER_CORS_ORIGINS="https://app.example.com" (koma = beberapa origin).
func runHTTPServer(addr string) {
	if addr == "" {
		addr = defaultHTTPAddr
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/scrape", handleAPIScrape)
	mux.HandleFunc("/api/scrape/status", handleScrapeStatus)

	handler := withCORS(mux)
	log.Printf("HTTP API: http://%s  | POST /api/scrape  GET /api/scrape/status?jobId=...\n", addr)
	log.Printf("FE dev: PUBLIC_API_URL=http://%s npm run dev\n", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": "maps-scraper"})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if corsAllowOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// corsAllowOrigin: jika SCRAPER_CORS_ORIGINS di-set, hanya origin yang terdaftar; jika kosong, izinkan localhost dev (semua port).
func corsAllowOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	if raw := strings.TrimSpace(os.Getenv("SCRAPER_CORS_ORIGINS")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if strings.TrimSpace(p) == origin {
				return true
			}
		}
		return false
	}
	return isLocalDevOrigin(origin)
}

func isLocalDevOrigin(o string) bool {
	for _, prefix := range []string{
		"http://localhost:",
		"http://127.0.0.1:",
		"http://[::1]:",
	} {
		if strings.HasPrefix(o, prefix) {
			return true
		}
	}
	return false
}
