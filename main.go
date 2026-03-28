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
	"strings"
	"time"
)

const (
	baseMapsURL = "https://www.google.com/maps/search/"
	defaultZoom = "17z"
	maxResults  = 50
)

func main() {
	keyword, locationName := getKeywordAndLocationName()
	if keyword == "" {
		log.Fatalf("❌ Keyword tidak boleh kosong.\n")
	}
	if locationName == "" {
		log.Fatalf("❌ Nama lokasi wajib diisi (contoh: jakarta, Leuwiliang).\n")
	}

	log.Printf("📍 Mencari koordinat untuk: %s ...\n", locationName)
	lat, lng, err := geocodeLocation(locationName)
	if err != nil {
		log.Fatalf("❌ Geocoding gagal: %v\n", err)
	}
	log.Printf("📍 Koordinat: %s, %s\n", lat, lng)

	searchURL := buildSearchURL(keyword, lat, lng)
	log.Printf("🔍 Keyword: %s\n", keyword)
	log.Printf("📍 URL: %s\n", searchURL)

	scraper := controllers.NewGoogleMapsScraper()
	defer scraper.Close()

	if err := scraper.Init(); err != nil {
		log.Fatalf("❌ Error initializing: %v\n", err)
	}

	stores, err := scraper.ScrapeCoffeeShops(searchURL, maxResults)
	if err != nil {
		log.Fatalf("❌ Error scraping: %v\n", err)
	}

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

func getKeywordAndLocationName() (keyword, locationName string) {
	args := os.Args[1:]
	if len(args) >= 2 {
		keyword = strings.TrimSpace(strings.Join(args[:len(args)-1], " "))
		locationName = strings.TrimSpace(args[len(args)-1])
		if keyword != "" && locationName != "" {
			return keyword, locationName
		}
	}
	reader := bufio.NewReader(os.Stdin)
	if len(args) == 1 {
		keyword = strings.TrimSpace(args[0])
	} else if len(args) == 0 {
		log.Print("🔑 Masukkan keyword pencarian (contoh: coffee shop, restoran): ")
		kwLine, _ := reader.ReadString('\n')
		keyword = strings.TrimSpace(kwLine)
	}
	for {
		log.Print("📍 Nama lokasi (contoh: jakarta, Leuwiliang, Bogor): ")
		locLine, _ := reader.ReadString('\n')
		locationName = strings.TrimSpace(locLine)
		if locationName != "" {
			return keyword, locationName
		}
		log.Println("⚠️  Nama lokasi tidak boleh kosong.")
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
