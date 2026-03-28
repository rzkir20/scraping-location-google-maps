package controllers

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strings"

	"location/types"
)

func (g *GoogleMapsScraper) SaveToFile(stores []types.StoreInfo, filename string) error {
	if filename == "" {
		filename = "results.json"
	}

	data, err := json.MarshalIndent(stores, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	log.Printf("💾 Results saved to: %s\n", filename)
	return nil
}

func (g *GoogleMapsScraper) SaveToCSV(stores []types.StoreInfo, filename string) error {
	if filename == "" {
		filename = "results.csv"
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"Nama Toko", "Phone Number"}); err != nil {
		return err
	}

	written := 0
	skippedNoPhone := 0
	for _, store := range stores {
		phone := strings.TrimSpace(store.Phone)
		if phone == "" {
			skippedNoPhone++
			continue
		}
		if err := writer.Write([]string{store.Name, phone}); err != nil {
			return err
		}
		written++
	}

	if skippedNoPhone > 0 {
		log.Printf("💾 CSV saved to: %s (%d baris; %d tanpa nomor tidak dimasukkan)\n", filename, written, skippedNoPhone)
	} else {
		log.Printf("💾 CSV saved to: %s\n", filename)
	}
	return nil
}
