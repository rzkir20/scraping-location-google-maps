package controllers

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strings"

	"location/types"
)

// sanitizeCSVField membersihkan teks agar aman untuk CSV/Excel: UTF-8 valid, tanpa NUL, baris baru diratakan.
func sanitizeCSVField(s string) string {
	s = strings.ToValidUTF8(strings.TrimSpace(s), "")
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0:
			continue
		case r == '\r':
			continue
		case r == '\n', r == '\t':
			b.WriteByte(' ')
		case r < 32:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// WriteStoresJSON menulis hasil ke writer (mis. dialog Simpan file di GUI).
func WriteStoresJSON(w io.Writer, stores []types.StoreInfo) error {
	data, err := json.MarshalIndent(stores, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// WriteStoresCSV menulis CSV ke writer; baris tanpa nomor telepon dilewati.
func WriteStoresCSV(w io.Writer, stores []types.StoreInfo) (written int, err error) {
	writer := csv.NewWriter(w)

	if err := writer.Write([]string{"Nama Toko", "Phone Number", "Alamat"}); err != nil {
		return 0, err
	}

	for _, store := range stores {
		phone := sanitizeCSVField(store.Phone)
		if phone == "" {
			continue
		}
		name := sanitizeCSVField(store.Name)
		if name == "" {
			name = "(tanpa nama)"
		}
		row := []string{name, phone, sanitizeCSVField(store.Address)}
		if err := writer.Write(row); err != nil {
			return written, err
		}
		written++
	}
	writer.Flush()
	return written, writer.Error()
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

	written, err := WriteStoresCSV(file, stores)
	if err != nil {
		return err
	}

	g.progressf("💾 Disimpan: %s (CSV, %d baris)", filename, written)
	return nil
}
