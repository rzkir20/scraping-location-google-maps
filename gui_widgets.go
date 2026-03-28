package main

import (
	"image"
	"image/color"

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
