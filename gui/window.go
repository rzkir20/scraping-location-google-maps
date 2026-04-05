package gui

import (
	"bytes"
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

// ScrapeFunc adalah callback ke logika scrape dari package main (tidak bisa import main dari sini).
type ScrapeFunc func(keyword, locationName string, maxResults int, logf func(string), logStores bool) ([]types.StoreInfo, error)

// RunGUI menjalankan loop Gio di goroutine terpisah lalu app.Main().
func RunGUI(runScrape ScrapeFunc, defaultMax int) {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Google Maps Scraper"),
			app.Size(unit.Dp(680), unit.Dp(820)),
		)
		if err := runGioWindow(w, runScrape, defaultMax); err != nil {
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
		var buf bytes.Buffer
		if err := write(&buf); err != nil {
			appendLogLine(fmt.Sprintf("Menulis %s gagal: %v\n", filename, err))
			return
		}
		if _, err := wc.Write(buf.Bytes()); err != nil {
			appendLogLine(fmt.Sprintf("Menyimpan %s ke file gagal: %v\n", filename, err))
			return
		}
		appendLogLine(successMsg)
	}()
}

func runGioWindow(w *app.Window, runScrape ScrapeFunc, defaultMax int) error {
	exp := explorer.NewExplorer(w)
	var kwEd, locEd, tgtEd, logEd widget.Editor
	var dark bool
	var logMu sync.Mutex
	var pendingLines []string
	var resultMu sync.Mutex
	var resultRows []types.StoreInfo
	var busy atomic.Bool
	var pageScroll layout.List
	var tableScrollH widget.List
	var showRescrapeModal bool
	var start, btnRescrape, btnDownloadCSV, btnDownloadJSON, btnTheme widget.Clickable
	var btnModalUnduh, btnModalLanjut, btnModalBatal widget.Clickable
	var btnFollowGitHub, btnFollowTikTok, btnFollowIG widget.Clickable

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
		// Pakai Insert per baris (bukan SetText): Insert mengaktifkan scrollCaret sehingga
		// viewport mengikuti ke bawah. SetText memindahkan caret ke awal dan sering tidak
		// meng-scroll ke akhir pada editor read-only.
		for _, s := range lines {
			logEd.SetCaret(logEd.Len(), logEd.Len())
			logEd.Insert(s + "\n")
		}
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

			var beginScrape func()
			beginScrape = func() {
				kw := strings.TrimSpace(kwEd.Text())
				loc := strings.TrimSpace(locEd.Text())
				tgtStr := strings.TrimSpace(tgtEd.Text())
				var n int
				var err error
				if tgtStr == "" {
					n = defaultMax
				} else {
					n, err = strconv.Atoi(tgtStr)
				}
				if err != nil || n < 1 {
					logEd.SetCaret(logEd.Len(), logEd.Len())
					logEd.Insert("Isi target dengan angka bulat ≥ 1, atau kosongkan untuk default 10.\n")
					return
				}
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
					defer func() {
						if r := recover(); r != nil {
							resultMu.Lock()
							resultRows = nil
							resultMu.Unlock()
							appendLogLine(fmt.Sprintf("GAGAL (panic): %v\n", r))
						}
					}()
					stores, err := runScrape(keyword, location, target, appendLogLine, false)
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

			if btnTheme.Clicked(gtx) && !showRescrapeModal {
				dark = !dark
				w.Invalidate()
			}
			pal := paletteLight()
			if dark {
				pal = paletteDark()
			}
			th := modernTheme(pal)

			if showRescrapeModal {
				if btnModalBatal.Clicked(gtx) {
					showRescrapeModal = false
				}
				if btnModalUnduh.Clicked(gtx) {
					showRescrapeModal = false
					appendLogLine("Silakan unduh CSV atau JSON di bagian Hasil jika ingin menyimpan data saat ini sebelum mengosongkan form.\n")
				}
				if btnModalLanjut.Clicked(gtx) && !busy.Load() {
					showRescrapeModal = false
					resultMu.Lock()
					resultRows = nil
					resultMu.Unlock()
					logEd.SetText("")
					kwEd.SetText("")
					locEd.SetText("")
					tgtEd.SetText("")
					appendLogLine("Form dan hasil dikosongkan. Isi kembali lalu klik Mulai scraping.\n")
					w.Invalidate()
				}
			}

			if btnRescrape.Clicked(gtx) && !busy.Load() && !showRescrapeModal {
				resultMu.Lock()
				has := len(resultRows) > 0
				resultMu.Unlock()
				if has {
					showRescrapeModal = true
					w.Invalidate()
				}
			}

			if start.Clicked(gtx) && !busy.Load() && !showRescrapeModal {
				beginScrape()
			}

			if btnDownloadCSV.Clicked(gtx) && !busy.Load() && !showRescrapeModal {
				resultMu.Lock()
				rows := append([]types.StoreInfo(nil), resultRows...)
				resultMu.Unlock()
				if len(rows) == 0 {
					appendLogLine("Belum ada hasil — jalankan scraping dulu sebelum unduh CSV.\n")
				} else {
					exportViaDialog(exp, appendLogLine, "results.csv", func(w io.Writer) error {
						_, err := controllers.WriteStoresCSV(w, rows)
						return err
					}, "CSV disimpan ke file yang Anda pilih.\n")
				}
			}

			if btnDownloadJSON.Clicked(gtx) && !busy.Load() && !showRescrapeModal {
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

			if btnFollowGitHub.Clicked(gtx) {
				go func() { _ = openBrowser(FollowURLGitHub) }()
			}
			if btnFollowTikTok.Clicked(gtx) {
				go func() { _ = openBrowser(FollowURLTikTok) }()
			}
			if btnFollowIG.Clicked(gtx) {
				go func() { _ = openBrowser(FollowURLInstagram) }()
			}

			resultMu.Lock()
			tableData := append([]types.StoreInfo(nil), resultRows...)
			resultMu.Unlock()

			scrollContent := func(gtx layout.Context, _ int) layout.Dimensions {
				logMaxH := gtx.Dp(300)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return labelMuted(th, pal, "Kumpulkan bisnis lokal tanpa website yang punya nomor telepon di Maps. Gulir halaman ini; Chrome terbuka saat scraping.").Layout(gx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gx)
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return roundedCard(gx, pal.Card, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "1", "Form pencarian", "Lengkapi tiga field berikut.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Kata kunci bisnis", "Contoh: rental mobil, coffee shop, laundry", &kwEd, "rental mobil")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Wilayah atau kota", "Lokasi dipakai untuk titik di peta", &locEd, "Jakarta")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, pal, "Target jumlah listing", "Tanpa website dan dengan nomor telepon. Kosongkan = default 10; isi angka ≥ 1.", &tgtEd, "10")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtxBtn := gtx
											if busy.Load() || showRescrapeModal {
												gtxBtn = gtx.Disabled()
											}
											return contrastIconButton(th, &start, iconPlayScrape, "Mulai scraping")(gtxBtn)
										}),
										layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtxBtn := gtx
											if busy.Load() || len(tableData) == 0 || showRescrapeModal {
												gtxBtn = gtx.Disabled()
											}
											return contrastIconButton(th, &btnRescrape, iconRescrape, "Scraping ulang")(gtxBtn)
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
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !showRescrapeModal {
							return layout.Spacer{Height: unit.Dp(18)}.Layout(gtx)
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
							layout.Rigid(rescrapeConfirmPanel(th, pal, &btnModalUnduh, &btnModalLanjut, &btnModalBatal)),
							layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return roundedCard(gx, pal.LogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "2", "Log aktivitas", "Proses scraping (buka Maps, kartu, ringkasan) — sama seperti log di terminal.")),
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
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return layout.Spacer{Height: unit.Dp(18)}.Layout(gx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gx := gtx
						if showRescrapeModal {
							gx = gtx.Disabled()
						}
						return roundedCard(gx, pal.LogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, pal, "3", "Hasil (tabel)", "Nama, telepon, alamat. Gulir horizontal jika tabel lebih lebar dari jendela.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return resultsTable(th, pal, tableData, &tableScrollH)(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(labelMuted(th, pal, "Unduh ke folder pilihan (dialog Simpan). Hanya listing yang punya nomor telepon.").Layout),
										layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtxDisabled := gtx
											if busy.Load() || len(tableData) == 0 || showRescrapeModal {
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
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gx := gtx
							if showRescrapeModal {
								gx = gtx.Disabled()
							}
							return stickyAppBar(th, pal, dark, &btnTheme)(gx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return pageScroll.Layout(gtx, 1, scrollContent)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return followUsFooter(th, pal, &btnFollowGitHub, &btnFollowTikTok, &btnFollowIG)(gtx)
						}),
					)
				})
			})

			e.Frame(gtx.Ops)
		}
	}
}
