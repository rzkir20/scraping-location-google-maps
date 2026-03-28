package main

import (
	"fmt"
	"image"
	"image/color"
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

	"location/types"
)

var (
	colPageBG    = color.NRGBA{R: 241, G: 245, B: 249, A: 255}
	colCard      = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colInputFill = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
	colLogCard   = color.NRGBA{R: 235, G: 241, B: 248, A: 255}
	colTitle     = color.NRGBA{R: 15, G: 23, B: 42, A: 255}
	colBody      = color.NRGBA{R: 30, G: 41, B: 59, A: 255}
	colMuted     = color.NRGBA{R: 100, G: 116, B: 139, A: 255}
	colAccent    = color.NRGBA{R: 13, G: 148, B: 136, A: 255}
	colAccentFg  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colTblHead   = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
	colTblRow    = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colTblRowAlt = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
)

func modernTheme() *material.Theme {
	th := material.NewTheme()
	th.Palette = material.Palette{
		Bg:         colCard,
		Fg:         colBody,
		ContrastBg: colAccent,
		ContrastFg: colAccentFg,
	}
	return th
}

func roundedCard(gtx layout.Context, bg color.NRGBA, corner unit.Dp, inner layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			r := gtx.Dp(corner)
			defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
			paint.ColorOp{Color: bg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: sz}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(18)).Layout(gtx, inner)
		},
	)
}

func pageFill(gtx layout.Context, bg color.NRGBA, child layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			defer clip.Rect{Max: sz}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: bg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: sz}
		},
		child,
	)
}

func labelMuted(th *material.Theme, txt string) material.LabelStyle {
	l := material.Body2(th, txt)
	l.Color = colMuted
	l.TextSize = th.TextSize * 85 / 100
	return l
}

func sectionTitle(th *material.Theme, number, title, subtitle string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				side := gtx.Dp(30)
				sz := image.Pt(side, side)
				num := material.Caption(th, number)
				num.Color = colAccentFg
				num.TextSize = unit.Sp(12)
				return layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						defer clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(10)).Push(gtx.Ops).Pop()
						paint.ColorOp{Color: colAccent}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: sz}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, num.Layout)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						t := material.Body1(th, title)
						t.Color = colTitle
						t.TextSize = unit.Sp(15)
						return t.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelMuted(th, subtitle).Layout(gtx)
					}),
				)
			}),
		)
	}
}

func fieldRow(th *material.Theme, label, hint string, editor *widget.Editor, placeholder string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(th, label)
				l.Color = colBody
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labelMuted(th, hint).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Background{}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Constraints.Min
						r := gtx.Dp(10)
						defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
						paint.ColorOp{Color: colInputFill}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: sz}
					},
					func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(12)).Layout(gtx,
							material.Editor(th, editor, placeholder).Layout,
						)
					},
				)
			}),
		)
	}
}

func normalizePhoneDisplay(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "tel:")
	s = strings.TrimPrefix(s, "Tel:")
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func tableCell(th *material.Theme, txt string, bold bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		in := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}
		return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(th, txt)
			l.Color = colBody
			if bold {
				l.Color = colTitle
			}
			return l.Layout(gtx)
		})
	}
}

func tableHeaderBar(th *material.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				defer clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(8)).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: colTblHead}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(40)
						gtx.Constraints.Max.X = gtx.Dp(52)
						return tableCell(th, "#", true)(gtx)
					}),
					layout.Flexed(1, tableCell(th, "Nama usaha", true)),
					layout.Flexed(1, tableCell(th, "Telepon", true)),
				)
			},
		)
	}
}

func tableDataRow(th *material.Theme, idx int, name, phone string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		bg := colTblRow
		if idx%2 == 1 {
			bg = colTblRowAlt
		}
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				defer clip.Rect{Max: sz}.Push(gtx.Ops).Pop()
				paint.ColorOp{Color: bg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(40)
						gtx.Constraints.Max.X = gtx.Dp(52)
						return tableCell(th, fmt.Sprintf("%d", idx), false)(gtx)
					}),
					layout.Flexed(1, tableCell(th, name, false)),
					layout.Flexed(1, tableCell(th, phone, false)),
				)
			},
		)
	}
}

func resultsTable(th *material.Theme, rows []types.StoreInfo) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if len(rows) == 0 {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx,
				labelMuted(th, "Tabel akan terisi setelah scraping selesai.").Layout,
			)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(tableHeaderBar(th)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ch := []layout.FlexChild{layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout)}
				for i, r := range rows {
					i := i
					r := r
					ch = append(ch, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return tableDataRow(th, i+1, r.Name, normalizePhoneDisplay(r.Phone))(gtx)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, ch...)
			}),
		)
	}
}

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

func runGioWindow(w *app.Window) error {
	th := modernTheme()
	var kwEd, locEd, tgtEd, logEd widget.Editor
	var start widget.Clickable
	var logMu sync.Mutex
	var pendingLines []string
	var resultMu sync.Mutex
	var resultRows []types.StoreInfo
	var busy atomic.Bool
	var pageScroll layout.List

	kwEd.SingleLine = true
	locEd.SingleLine = true
	tgtEd.SingleLine = true
	tgtEd.SetText("10")
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
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			flushPendingLog()

			if start.Clicked(gtx) && !busy.Load() {
				kw := strings.TrimSpace(kwEd.Text())
				loc := strings.TrimSpace(locEd.Text())
				n, err := strconv.Atoi(strings.TrimSpace(tgtEd.Text()))
				if err != nil || n < 1 {
					logEd.SetText(logEd.Text() + "Isi target dengan angka bulat ≥ 1.\n")
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
						logf := func(s string) {
							logMu.Lock()
							pendingLines = append(pendingLines, s)
							logMu.Unlock()
							w.Invalidate()
						}
						stores, err := runScrapeJob(keyword, location, target, logf, false)
						resultMu.Lock()
						if err == nil && stores != nil {
							resultRows = append([]types.StoreInfo(nil), stores...)
						} else {
							resultRows = nil
						}
						resultMu.Unlock()
						if err != nil {
							logf(fmt.Sprintf("GAGAL: %v", err))
						} else {
							logf("Selesai. Cek results.json dan results.csv di folder aplikasi.")
						}
						w.Invalidate()
					}(kw, loc, n)
				}
			}

			resultMu.Lock()
			tableData := append([]types.StoreInfo(nil), resultRows...)
			resultMu.Unlock()

			scrollContent := func(gtx layout.Context, _ int) layout.Dimensions {
				logMaxH := gtx.Dp(220)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						h := material.H4(th, "Google Maps Scraper")
						h.Color = colTitle
						return h.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(labelMuted(th, "Kumpulkan nama & telepon bisnis di Maps yang terlihat belum punya website sendiri.").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(labelMuted(th, "Gulir halaman dengan mouse / trackpad. Chrome akan terbuka saat scraping.").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return roundedCard(gtx, colCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, "1", "Form pencarian", "Lengkapi tiga field berikut.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
								layout.Rigid(fieldRow(th, "Kata kunci bisnis", "Contoh: rental mobil, coffee shop, laundry", &kwEd, "rental mobil")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, "Wilayah atau kota", "Lokasi dipakai untuk titik di peta", &locEd, "Jakarta")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(fieldRow(th, "Target jumlah listing", "Berapa listing tanpa website yang ingin dikumpulkan (≥ 1)", &tgtEd, "10")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := material.Button(th, &start, "Mulai scraping")
											if busy.Load() {
												gtx = gtx.Disabled()
											}
											return btn.Layout(gtx)
										}),
										layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if !busy.Load() {
												return layout.Dimensions{}
											}
											return labelMuted(th, "Memproses… mohon tunggu, jangan tutup Chrome jika sedang dipakai.").Layout(gtx)
										}),
									)
								}),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return roundedCard(gtx, colLogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, "2", "Log aktivitas", "Teks langkah & file tersimpan.")),
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
											paint.ColorOp{Color: colCard}.Add(gtx.Ops)
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
						return roundedCard(gtx, colLogCard, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(sectionTitle(th, "3", "Hasil (tabel)", "Nama dan telepon per baris.")),
								layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
								layout.Rigid(resultsTable(th, tableData)),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
				)
			}

			pageScroll.Axis = layout.Vertical
			pageFill(gtx, colPageBG, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return pageScroll.Layout(gtx, 1, scrollContent)
				})
			})

			e.Frame(gtx.Ops)
		}
	}
}