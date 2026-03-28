package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"location/controllers"
	"location/types"
)

func runGUI() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Google Maps Scraper"),
			app.Size(unit.Dp(680), unit.Dp(820)),
		)
		if err := runGioWindow(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

// exportViaDialog menyimpan hasil lewat dialog Simpan file (CSV atau JSON).
func exportViaDialog(exp *explorer.Explorer, appendLogLine func(string), filename string, write func(io.Writer) error, successMsg string) {
	go func() {
		wc, err := exp.CreateFile(filename)
		if err != nil {
			if !errors.Is(err, explorer.ErrUserDecline) {
				appendLogLine(fmt.Sprintf("Unduh %s gagal: %v\n", filename, err))
			}
			return
		}
		defer wc.Close()
		if err := write(wc); err != nil {
			appendLogLine(fmt.Sprintf("Menulis %s gagal: %v\n", filename, err))
			return
		}
		appendLogLine(successMsg)
	}()
}

func runGioWindow(w *app.Window) error {
	exp := explorer.NewExplorer(w)
	var kwEd, locEd, tgtEd, logEd widget.Editor
	var start, btnDownloadCSV, btnDownloadJSON, btnTheme widget.Clickable
	var dark bool
	var logMu sync.Mutex
	var pendingLines []string
	var resultMu sync.Mutex
	var resultRows []types.StoreInfo
	var busy atomic.Bool
	var pageScroll layout.List

	kwEd.SingleLine = true
	locEd.SingleLine = true
	tgtEd.SingleLine = true
	logEd.ReadOnly = true

	flushPendingLog := func() {
		logMu.Lock()
		lines := pendingLines
		pendingLines = nil
		logMu.Unlock()
		if len(lines) == 0 {
			return
		}
		var b strings.Builder
		b.WriteString(logEd.Text())
		for _, s := range lines {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		logEd.SetText(b.String())
	}

	var ops op.Ops
	for {
		ev := w.Event()
		exp.ListenEvents(ev)
		switch e := ev.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			flushPendingLog()

			appendLogLine := func(s string) {
				logMu.Lock()
				pendingLines = append(pendingLines, s)
				logMu.Unlock()
				w.Invalidate()
			}

			if btnTheme.Clicked(gtx) {
				dark = !dark
				w.Invalidate()
			}
			pal := paletteLight()
			if dark {
				pal = paletteDark()
			}
			th := modernTheme(pal)

			if start.Clicked(gtx) && !busy.Load() {
				kw := strings.TrimSpace(kwEd.Text())
				loc := strings.TrimSpace(locEd.Text())
				tgtStr := strings.TrimSpace(tgtEd.Text())
				var n int
				var err error
				if tgtStr == "" {
					n = defaultMaxResults
				} else {
					n, err = strconv.Atoi(tgtStr)
				}
				if err != nil || n < 1 {
					logEd.SetText(logEd.Text() + "Isi target dengan angka bulat ≥ 1, atau kosongkan untuk default 10.\n")
				} else {
					logEd.SetText("")
					resultMu.Lock()
					resultRows = nil
					resultMu.Unlock()
					busy.Store(true)
					go func(keyword, location string, target int) {
						defer func() {
							busy.Store(false)
							w.Invalidate()
						}()
						stores, err := runScrapeJob(keyword, location, target, appendLogLine, false)
						resultMu.Lock()
						if err == nil && stores != nil {
							resultRows = append([]types.StoreInfo(nil), stores...)
						} else {
							resultRows = nil
						}
						resultMu.Unlock()
						if err != nil {
							appendLogLine(fmt.Sprintf("GAGAL: %v", err))
						} else {
							appendLogLine("Selesai. File juga tersimpan sebagai results.json & results.csv di folder aplikasi, atau unduh lewat tombol di bagian Hasil.")
						}
					}(kw, loc, n)
				}
			}

			if btnDownloadCSV.Clicked(gtx) && !busy.Load() {
				resultMu.Lock()
				rows := append([]types.StoreInfo(nil), resultRows...)
				resultMu.Unlock()
				if len(rows) == 0 {
					appendLogLine("Belum ada hasil — jalankan scraping dulu sebelum unduh CSV.\n")
				} else {
					exportViaDialog(exp, appendLogLine, "results.csv", func(w io.Writer) error {
						_, _, err := controllers.WriteStoresCSV(w, rows)
						return err
					}, "CSV disimpan ke file yang Anda pilih.\n")
				}
			}

			if btnDownloadJSON.Clicked(gtx) && !busy.Load() {
				resultMu.Lock()
				rows := append([]types.StoreInfo(nil), resultRows...)
				resultMu.Unlock()
				if len(rows) == 0 {
					appendLogLine("Belum ada hasil — jalankan scraping dulu sebelum unduh JSON.\n")
				} else {
					exportViaDialog(exp, appendLogLine, "results.json", func(w io.Writer) error {
						return controllers.WriteStoresJSON(w, rows)
					}, "JSON disimpan ke file yang Anda pilih.\n")
				}
			}

			resultMu.Lock()
			tableData := append([]types.StoreInfo(nil), resultRows...)
			resultMu.Unlock()

			scrollContent := func(gtx layout.Context, _ int) layout.Dimensions {
				logMaxH := gtx.Dp(220)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(labelMuted(th, pal, "Kumpulkan nama & telepon bisnis lokal tanpa website. Gulir halaman ini; Chrome terbuka saat scraping.").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return roundedCard(gtx, pal.Card, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "1", "Form pencarian", "Lengkapi tiga field berikut.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Kata kunci bisnis", "Contoh: rental mobil, coffee shop, laundry", &kwEd, "rental mobil")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Wilayah atau kota", "Lokasi dipakai untuk titik di peta", &locEd, "Jakarta")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Target jumlah listing", "Kosongkan untuk pakai default 10; isi angka ≥ 1 jika ingin lain.", &tgtEd, "10")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtxBtn := gtx
											if busy.Load() {
												gtxBtn = gtx.Disabled()
											}
											return contrastIconButton(th, &start, iconPlayScrape, "Mulai scraping")(gtxBtn)
										}),
										layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if !busy.Load() {
												return layout.Dimensions{}
											}
											return labelMuted(th, pal, "Memproses… mohon tunggu, jangan tutup Chrome jika sedang dipakai.").Layout(gtx)
										}),
									)
								}),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return roundedCard(gtx, pal.LogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "2", "Log aktivitas", "Teks langkah & file tersimpan.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									mh := gtx.Constraints.Max.Y
									if mh > logMaxH {
										mh = logMaxH
									}
									if mh < gtx.Dp(100) {
										mh = gtx.Dp(100)
									}
									gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, mh))
									return layout.Background{}.Layout(gtx,
										func(gtx layout.Context) layout.Dimensions {
											sz := gtx.Constraints.Min
											r := gtx.Dp(10)
											defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
											paint.ColorOp{Color: pal.EditorBg}.Add(gtx.Ops)
											paint.PaintOp{}.Add(gtx.Ops)
											return layout.Dimensions{Size: sz}
										},
										func(gtx layout.Context) layout.Dimensions {
											return layout.UniformInset(unit.Dp(12)).Layout(gtx,
												material.Editor(th, &logEd, "Log muncul di sini…").Layout,
											)
										},
									)
								}),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return roundedCard(gtx, pal.LogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "3", "Hasil (tabel)", "Nama dan telepon per baris.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
								layout.Rigid(resultsTable(th, pal, tableData)),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(labelMuted(th, pal, "Unduh ke folder pilihan (dialog Simpan). CSV hanya baris ber nomor.").Layout),
										layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtxDisabled := gtx
											if busy.Load() || len(tableData) == 0 {
												gtxDisabled = gtx.Disabled()
											}
											return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return contrastIconButton(th, &btnDownloadCSV, iconFileCSV, "Unduh CSV")(gtxDisabled)
												}),
												layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return contrastIconButton(th, &btnDownloadJSON, iconFileJSON, "Unduh JSON")(gtxDisabled)
												}),
											)
										}),
									)
								}),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
				)
			}

			pageScroll.Axis = layout.Vertical
			pageFill(gtx, pal.PageBG, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(stickyAppBar(th, pal, dark, &btnTheme)),
						layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return pageScroll.Layout(gtx, 1, scrollContent)
						}),
					)
				})
			})

			e.Frame(gtx.Ops)
		}
	}
}
