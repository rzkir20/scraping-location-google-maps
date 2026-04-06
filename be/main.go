package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

)

const (
	baseMapsURL       = "https://www.google.com/maps/search/"
	defaultZoom       = "17z"
	defaultMaxResults = 10
)

func main() {
	// Mode API untuk FE: go run . server   atau   SCRAPER_HTTP_ADDR=127.0.0.1:8080 go run .
	if addr := strings.TrimSpace(os.Getenv("SCRAPER_HTTP_ADDR")); addr != "" && len(os.Args) == 1 {
		runHTTPServer(addr)
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "server" {
		addr := defaultHTTPAddrFromEnv()
		if len(os.Args) >= 3 && strings.TrimSpace(os.Args[2]) != "" {
			addr = strings.TrimSpace(os.Args[2])
		}
		runHTTPServer(addr)
		return
	}
	runCLI()
}

func runCLI() {
	keyword, locationName, maxResults := getKeywordLocationAndTarget()
	logf := func(s string) { log.Println(s) }
	if _, err := RunScrapeJob(keyword, locationName, maxResults, logf, true, ScrapeJobOptions{}); err != nil {
		log.Fatalf("❌ %v\n", err)
	}
}

func getPhoneDisplay(phone string) string {
	if phone == "" {
		return "N/A"
	}
	return phone
}

// getKeywordLocationAndTarget urutan: nama/keyword → lokasi → target jumlah listing.
// CLI: argumen terakhir angka = target; sebelum itu = lokasi; sisanya = keyword.
// Contoh: go run main.go rental mobil bandung 30
func getKeywordLocationAndTarget() (keyword, locationName string, maxResults int) {
	args := os.Args[1:]
	reader := bufio.NewReader(os.Stdin)

	if len(args) >= 3 {
		last := strings.TrimSpace(args[len(args)-1])
		if n, err := strconv.Atoi(last); err == nil && n > 0 {
			maxResults = n
			locationName = strings.TrimSpace(args[len(args)-2])
			keyword = strings.TrimSpace(strings.Join(args[:len(args)-2], " "))
			if keyword != "" && locationName != "" {
				return keyword, locationName, maxResults
			}
		}
	}
	if len(args) == 2 {
		keyword = strings.TrimSpace(args[0])
		locationName = strings.TrimSpace(args[1])
		maxResults = readTargetListing(reader, defaultMaxResults)
		return keyword, locationName, maxResults
	}
	if len(args) == 1 {
		keyword = strings.TrimSpace(args[0])
	} else if len(args) == 0 {
		log.Print("🔑 Nama / keyword pencarian (contoh: coffee shop, rental mobil): ")
		kwLine, _ := reader.ReadString('\n')
		keyword = strings.TrimSpace(kwLine)
	} else {
		log.Println("⚠️  Format CLI: <keyword …> <lokasi> <angka_target>  Contoh: rental mobil bandung 30")
		log.Print("🔑 Nama / keyword pencarian (contoh: coffee shop, rental mobil): ")
		kwLine, _ := reader.ReadString('\n')
		keyword = strings.TrimSpace(kwLine)
	}
	for {
		log.Print("📍 Nama lokasi (contoh: Jakarta, Bandung, Bogor): ")
		locLine, _ := reader.ReadString('\n')
		locationName = strings.TrimSpace(locLine)
		if locationName != "" {
			break
		}
		log.Println("⚠️  Nama lokasi tidak boleh kosong.")
	}
	maxResults = readTargetListing(reader, defaultMaxResults)
	return keyword, locationName, maxResults
}

func readTargetListing(reader *bufio.Reader, fallback int) int {
	for {
		log.Printf("🎯 Berapa listing maksimal (tanpa website, wajib ada nomor HP)? [default %d, Enter = default]: ", fallback)
		line, _ := reader.ReadString('\n')
		s := strings.TrimSpace(line)
		if s == "" {
			return fallback
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			log.Println("⚠️  Masukkan angka bulat positif, atau Enter untuk default.")
			continue
		}
		return n
	}
}

func geocodeLocation(placeName string) (lat, lng string, err error) {
	u := "https://nominatim.openstreetmap.org/search?q=" + url.QueryEscape(placeName) + "&format=json&limit=1"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "LocationScraper/1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return "", "", err
	}
	if len(results) == 0 {
		return "", "", fmt.Errorf("lokasi tidak ditemukan: %s", placeName)
	}
	return results[0].Lat, results[0].Lon, nil
}

func buildSearchURL(keyword, lat, lng string) string {
	encoded := url.PathEscape(keyword)
	return baseMapsURL + encoded + "/@" + lat + "," + lng + "," + defaultZoom + "/data=!3m1!4b1?entry=ttu"
}
