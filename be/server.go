package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

func defaultHTTPAddrFromEnv() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	// Listen on all interfaces by default.
	return ":" + port
}

// runHTTPServer mendengarkan di addr (mis. "127.0.0.1:8080") untuk dipanggil dari FE lewat fetch.
// CORS dev: semua origin http://localhost:* , http://127.0.0.1:* , http://[::1]:* (port bebas, cocok Astro 4321/5173/dll).
// Produksi: set CORS_ALLOW_ORIGINS="https://app.example.com" (koma = beberapa origin).
func runHTTPServer(addr string) {
	if addr == "" {
		addr = defaultHTTPAddrFromEnv()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/scrape", handleAPIScrape)
	mux.HandleFunc("/api/scrape/status", handleScrapeStatus)

	handler := withCORS(mux)
	displayURL := displayBaseURL(addr)
	log.Printf("HTTP API: %s  | POST /api/scrape  GET /api/scrape/status?jobId=...\n", displayURL)
	log.Printf("FE dev: PUBLIC_API_URL=%s npm run dev\n", displayURL)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func displayBaseURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
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
		w.Header().Add("Vary", "Origin")
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

// corsAllowOrigin: jika CORS_ALLOW_ORIGINS di-set, hanya origin yang terdaftar; jika kosong, izinkan localhost dev (semua port).
func corsAllowOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	// Selalu izinkan origin lokal untuk kebutuhan development lintas port.
	if isLocalDevOrigin(origin) {
		return true
	}
	if raw := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			v := strings.TrimSpace(p)
			if v == "*" || v == origin {
				return true
			}
		}
		return false
	}
	return false
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
