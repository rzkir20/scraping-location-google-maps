# Google Maps Scraper Bisnis

Scraper untuk mencari bisnis di Google Maps (keyword + lokasi), lalu mengumpulkan **nama** dan **nomor telepon** untuk tempat yang di panel Maps terlihat **belum/tidak punya situs web sendiri**.

**Versi Go** — satu binary, pakai Chrome lewat [chromedp](https://github.com/chromedp/chromedp).

## Instalasi

```bash
go mod download
```

## Build

Seluruh file dalam folder ikut dikompilasi; **jangan** hanya `main.go`:

```bash
go build -o scraper .
```

Tanpa jendela konsol (Windows): `go build -ldflags="-H windowsgui" -o scraper_gui.exe .`

## Menjalankan

### Mode aplikasi grafis (default)

Tanpa argumen — terbuka **jendela aplikasi** (bukan Chrome untuk UI; antarmuka pakai [Gio](https://gioui.org)). Isi keyword, lokasi, dan target, lalu **Mulai scraping**. Untuk mengambil data, **Google Chrome** tetap dibuka otomatis oleh scraper (chromedp), sama seperti sebelumnya.

```bash
go run .
```

Atau jalankan `scraper` / `scraper_gui.exe` tanpa argumen. Tutup jendela aplikasi jika sudah selesai.

### Mode interaktif (terminal)

Berikan **minimal satu argumen** (kalau tidak ada argumen, mode grafis yang jalan):

```bash
go run . "coffee shop"
```

Program akan meminta **berurutan**:

1. **Keyword / nama yang dicari** (misalnya: `coffee shop`, `rental mobil`)
2. **Lokasi** (misalnya: `Bandung`, `Jakarta`) — koordinat diambil lewat OpenStreetMap Nominatim
3. **Target** — berapa banyak listing **tanpa website** yang ingin dikumpulkan (angka ≥ 1). **Enter** = default **10**. Tidak ada batas maksimum dari aplikasi; target besar akan memakan waktu lebih lama.

### Mode CLI (tanpa prompt target jika lengkap)

Argumen terakhir harus **angka** (target), sebelum itu **lokasi**, sisanya **keyword**:

```bash
go run . rental mobil bandung 30
```

- Keyword: `rental mobil`
- Lokasi: `bandung`
- Target: `30`

Jika hanya dua argumen (`keyword` dan `lokasi`), target akan ditanya di terminal.

## Output

| File | Isi |
|------|-----|
| `results.json` | Semua listing yang lolos filter (nama + telepon jika ada). |
| `results.csv` | Kolom **Nama Toko** dan **Phone Number** — **hanya baris yang punya nomor telepon**; listing tanpa nomor tidak ditulis ke CSV. |

## Catatan

- Filter **tanpa website** mengikuti apa yang terlihat di panel detail Maps (bukan jaminan 100% akurat).
- Pastikan **Google Chrome** terpasang, atau set environment variable `CHROME_PATH` ke `chrome.exe` (lihat `controllers/maps_chrome.go`).
- Browser dibuka **non-headless** secara default. Untuk headless, ubah flag di `controllers/maps_chrome.go` (`chromedp.Flag("headless", true)`).
- Scraping mematuhi batasan dan perubahan UI Google Maps; hasil bisa bervariasi.

## Keuntungan Versi Go

- **Binary tunggal** — mudah dibagikan dan dijalankan
- **Performa native** — tanpa runtime JavaScript terpisah
- **Kontrol penuh** lewat Go + Chrome DevTools Protocol

## Ikuti & dukung

Kalau repo ini membantu atau kamu suka project-nya, jangan lupa kasih **Star** di GitHub.

- [GitHub — @rzkir20](https://github.com/rzkir20)
- [Instagram — @rzkir.20](https://www.instagram.com/rzkir.20/)
- [TikTok — @rzkir.20](https://www.tiktok.com/@rzkir.20)

---

## License

Source code berlisensi [MIT License](LICENSE) (Copyright © 2026 Rizki Ramadhan).
