package controllers

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strings"

	"location/types"
)

// WriteStoresJSON menulis hasil ke writer (mis. dialog Simpan file di GUI).
func WriteStoresJSON(w io.Writer, stores []types.StoreInfo) error {
	data, err := json.MarshalIndent(stores, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// WriteStoresCSV menulis CSV ke writer; baris tanpa nomor telepon dilewati (sama seperti SaveToCSV).
func WriteStoresCSV(w io.Writer, stores []types.StoreInfo) (written, skippedNoPhone int, err error) {
	writer := csv.NewWriter(w)

	if err := writer.Write([]string{"Nama Toko", "Phone Number", "Alamat"}); err != nil {
		return 0, 0, err
	}

	for _, store := range stores {
		phone := strings.TrimSpace(store.Phone)
		if phone == "" {
			skippedNoPhone++
			continue
		}
		if err := writer.Write([]string{store.Name, phone, strings.TrimSpace(store.Address)}); err != nil {
			return written, skippedNoPhone, err
		}
		written++
	}
	writer.Flush()
	return written, skippedNoPhone, writer.Error()
}

func (g *GoogleMapsScraper) SaveToFile(stores []types.StoreInfo, filename string) error {
	if filename == "" {
		filename = "results.json"
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := WriteStoresJSON(f, stores); err != nil {
		return err
	}
	g.progressf("💾 Disimpan: %s (JSON)", filename)
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

	written, skippedNoPhone, err := WriteStoresCSV(file, stores)
	if err != nil {
		return err
	}

	if skippedNoPhone > 0 {
		g.progressf("💾 Disimpan: %s (CSV, %d baris; %d tanpa nomor tidak dimasukkan)", filename, written, skippedNoPhone)
	} else {
		g.progressf("💾 Disimpan: %s (CSV)", filename)
	}
	return nil
}
