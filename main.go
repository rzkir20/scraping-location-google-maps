package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"location/controllers"
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
	keyword, locationName, maxResults := getKeywordLocationAndTarget()
	if keyword == "" {
		log.Fatalf("❌ Keyword tidak boleh kosong.\n")
	}
	if locationName == "" {
		log.Fatalf("❌ Nama lokasi wajib diisi (contoh: jakarta, Leuwiliang).\n")
	}
	if maxResults < 1 {
		log.Fatalf("❌ Target harus minimal 1.\n")
	}

	log.Printf("📍 Mencari koordinat untuk: %s ...\n", locationName)
	lat, lng, err := geocodeLocation(locationName)
	if err != nil {
		log.Fatalf("❌ Geocoding gagal: %v\n", err)
	}
	log.Printf("📍 Koordinat: %s, %s\n", lat, lng)

	searchURL := buildSearchURL(keyword, lat, lng)
	log.Printf("🔍 Keyword: %s\n", keyword)
	log.Printf("🎯 Target listing (tanpa website): %d\n", maxResults)
	log.Printf("📍 URL: %s\n", searchURL)

	scraper, err := controllers.NewGoogleMapsScraper()
	if err != nil {
		log.Fatalf("❌ Browser: %v\n", err)
	}
	defer scraper.Close()

	if err := scraper.Init(); err != nil {
		log.Fatalf("❌ Error initializing: %v\n", err)
	}

	stores, summary, err := scraper.ScrapeCoffeeShops(searchURL, maxResults)
	if err != nil {
		log.Fatalf("❌ Error scraping: %v\n", err)
	}

	log.Printf("\n📊 Total tersimpan: %d dari target maks. %d · dilewati (ada website): %d\n",
		len(stores), summary.TargetMax, summary.WithWebsite)
	log.Printf("\n📊 Total stores found: %d\n", len(stores))
	log.Println("\n📋 Results:")
	for i, store := range stores {
		log.Printf("%d. %s - Phone: %s\n", i+1, store.Name, getPhoneDisplay(store.Phone))
	}

	// Save results
	if err := scraper.SaveToFile(stores, "results.json"); err != nil {
		log.Printf("⚠️  Error saving JSON: %v\n", err)
	}

	if err := scraper.SaveToCSV(stores, "results.csv"); err != nil {
		log.Printf("⚠️  Error saving CSV: %v\n", err)
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
		log.Printf("🎯 Berapa listing maksimal yang ingin diambil (tanpa website)? [default %d, Enter = default]: ", fallback)
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
