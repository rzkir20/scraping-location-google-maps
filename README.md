# Google Maps Scraper Bisnis

Scraper untuk mencari bisnis di Google Maps yang belum memiliki website, kemudian mengambil nomor telepon dan nama toko.

**Versi Go** - Lebih cepat dan efisien dibanding versi TypeScript!

## Instalasi

```bash
go mod download
```

## Build

```bash
go build -o scraper main.go
```

## Menjalankan

```bash
go run main.go
```

atau jika sudah di-build:

```bash
./scraper
```

## Output

Hasil scraping akan disimpan dalam:

- `results.json` - Format JSON
- `results.csv` - Format CSV

## Catatan

- Scraper akan otomatis memfilter coffee shop yang **tidak memiliki website**
- Hasil akan mencakup nama toko dan nomor telepon (jika tersedia)
- Pastikan koneksi internet stabil saat menjalankan scraper
- Browser akan terbuka secara otomatis (headless: false). Untuk production, ubah ke `true` di `controllers/maps_controller.go`

## Keuntungan Versi Go

- ⚡ **Lebih cepat** - Kompilasi native, tidak perlu runtime
- 🚀 **Lebih efisien** - Menggunakan goroutines untuk concurrency
- 💪 **Lebih ringan** - Memory footprint lebih kecil
- 🔧 **Lebih mudah deploy** - Single binary executable

## Ikuti & dukung

Kalau repo ini membantu atau kamu suka project-nya, jangan lupa kasih **Star** di GitHub.

- [GitHub — @rzkir20](https://github.com/rzkir20)
- [Instagram — @rzkir.20](https://www.instagram.com/rzkir.20/)
- [TikTok — @rzkir.20](https://www.tiktok.com/@rzkir.20)

---

## License

Source code berlisensi [MIT License](LICENSE) (Copyright © 2026 Rizki Ramadhan).
# scraping-location-google-maps
