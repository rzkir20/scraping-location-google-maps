package gui

import (
	"image"
	"image/color"
	"os/exec"
	"runtime"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

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

// contrastIconButton tombol material (ikon + teks) dengan warna kontras pada aksen.
func contrastIconButton(th *material.Theme, clk *widget.Clickable, ic *widget.Icon, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return material.ButtonLayout(th, clk).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Dp(20)
						gtx.Constraints = layout.Exact(image.Pt(sz, sz))
						return ic.Layout(gtx, th.Palette.ContrastFg)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(th, label)
						l.Color = th.Palette.ContrastFg
						l.TextSize = th.TextSize * 14.0 / 16.0
						return l.Layout(gtx)
					}),
				)
			})
		})
	}
}

// stickyAppBar header tetap di atas saat konten di-scroll (dipisah dari layout.List).
func stickyAppBar(th *material.Theme, p uiPalette, dark bool, btnTheme *widget.Clickable) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				r := gtx.Dp(14)
				defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: p.StickyBar}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											sz := gtx.Dp(28)
											gtx.Constraints = layout.Exact(image.Pt(sz, sz))
											return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return iconMapsMap.Layout(gtx, p.Accent)
											})
										}),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													h := material.H5(th, "Google Maps Scraper")
													h.Color = p.Title
													return h.Layout(gtx)
												}),
												layout.Rigid(labelMuted(th, p, "Tanpa website · Maps").Layout),
											)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									ic := iconThemeSun
									if dark {
										ic = iconThemeDark
									}
									txt := "Tema gelap"
									if dark {
										txt = "Tema terang"
									}
									return contrastIconButton(th, btnTheme, ic, txt)(gtx)
								}),
							)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						w := gtx.Constraints.Max.X
						if w < gtx.Constraints.Min.X {
							w = gtx.Constraints.Min.X
						}
						h := gtx.Dp(1)
						gtx.Constraints = layout.Exact(image.Pt(w, h))
						defer clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: p.StickySep}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: image.Pt(w, h)}
					}),
				)
			},
		)
	}
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

func labelMuted(th *material.Theme, p uiPalette, txt string) material.LabelStyle {
	l := material.Body2(th, txt)
	l.Color = p.Muted
	l.TextSize = th.TextSize * 85 / 100
	return l
}

func sectionTitle(th *material.Theme, p uiPalette, number, title, subtitle string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				side := gtx.Dp(30)
				sz := image.Pt(side, side)
				num := material.Caption(th, number)
				num.Color = p.AccentFg
				num.TextSize = unit.Sp(12)
				return layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						defer clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(10)).Push(gtx.Ops).Pop()
						paint.ColorOp{Color: p.Accent}.Add(gtx.Ops)
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
						t.Color = p.Title
						t.TextSize = unit.Sp(15)
						return t.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelMuted(th, p, subtitle).Layout(gtx)
					}),
				)
			}),
		)
	}
}

// rescrapeConfirmPanel konfirmasi inline di alur scroll (tanpa overlay modal).
func rescrapeConfirmPanel(th *material.Theme, p uiPalette,
	btnUnduh, btnLanjut, btnBatal *widget.Clickable,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				r := gtx.Dp(14)
				defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: color.NRGBA{R: 30, G: 41, B: 59, A: 55}}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return roundedCard(gtx, p.Card, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								h := material.H6(th, "Isi form baru?")
								h.Color = p.Title
								return h.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
							layout.Rigid(labelMuted(th, p, "Hasil dan log akan dikosongkan, lalu form pencarian ditampilkan lagi. Unduh CSV atau JSON dulu jika perlu menyimpan data.").Layout),
							layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
							layout.Rigid(labelMuted(th, p, "Isi ulang tidak menjalankan scraping — setelah form muncul, klik Mulai scraping.").Layout),
							layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
									layout.Flexed(1, material.Button(th, btnUnduh, "Unduh dulu").Layout),
									layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
									layout.Flexed(1, material.Button(th, btnLanjut, "Isi ulang").Layout),
								)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Stack{Alignment: layout.Center}.Layout(gtx,
									layout.Stacked(material.Button(th, btnBatal, "Batal").Layout),
								)
							}),
						)
					})
				})
			},
		)
	}
}

func fieldRow(th *material.Theme, p uiPalette, label, hint string, editor *widget.Editor, placeholder string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(th, label)
				l.Color = p.Body
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labelMuted(th, p, hint).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Background{}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Constraints.Min
						r := gtx.Dp(10)
						defer clip.UniformRRect(image.Rectangle{Max: sz}, r).Push(gtx.Ops).Pop()
						paint.ColorOp{Color: p.InputFill}.Add(gtx.Ops)
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

// URL profil untuk footer "Ikuti kami" (selaras dengan README).
const (
	FollowURLGitHub    = "https://github.com/rzkir20"
	FollowURLTikTok    = "https://www.tiktok.com/@rzkir.20"
	FollowURLInstagram = "https://www.instagram.com/rzkir.20/"
)

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// followUsFooter baris bawah seperti footer: judul + tautan ke media sosial.
func followUsFooter(th *material.Theme, p uiPalette, gh, tiktok, ig *widget.Clickable) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Constraints.Max.X
				if w < gtx.Constraints.Min.X {
					w = gtx.Constraints.Min.X
				}
				h := gtx.Dp(1)
				gtx.Constraints = layout.Exact(image.Pt(w, h))
				defer clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops).Pop()
				paint.ColorOp{Color: p.StickySep}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: image.Pt(w, h)}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Stacked(labelMuted(th, p, "Ikuti kami").Layout),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, followLink(th, gh, iconFollowGH, "GitHub")),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, followLink(th, tiktok, iconFollowTikTok, "TikTok")),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, followLink(th, ig, iconFollowIG, "Instagram")),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Stacked(labelMuted(th, p, "Tautan dibuka di browser default.").Layout),
				)
			}),
		)
	}
}

func followLink(th *material.Theme, clk *widget.Clickable, ic *widget.Icon, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return material.ButtonLayout(th, clk).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Dp(20)
						gtx.Constraints = layout.Exact(image.Pt(sz, sz))
						return ic.Layout(gtx, th.Palette.ContrastFg)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(th, label)
						l.Color = th.Palette.ContrastFg
						l.TextSize = th.TextSize * 14.0 / 16.0
						return l.Layout(gtx)
					}),
				)
			})
		})
	}
}
