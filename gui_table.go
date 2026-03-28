package main

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"location/types"
)

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

func tableCell(th *material.Theme, p uiPalette, txt string, bold bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		in := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}
		return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(th, txt)
			l.Color = p.Body
			if bold {
				l.Color = p.Title
			}
			return l.Layout(gtx)
		})
	}
}

func tableHeaderBar(th *material.Theme, p uiPalette) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				defer clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(8)).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: p.TblHead}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(40)
						gtx.Constraints.Max.X = gtx.Dp(52)
						return tableCell(th, p, "#", true)(gtx)
					}),
					layout.Flexed(1, tableCell(th, p, "Nama usaha", true)),
					layout.Flexed(1, tableCell(th, p, "Telepon", true)),
				)
			},
		)
	}
}

func tableDataRow(th *material.Theme, p uiPalette, idx int, name, phone string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		bg := p.TblRow
		if idx%2 == 1 {
			bg = p.TblRowAlt
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
						return tableCell(th, p, fmt.Sprintf("%d", idx), false)(gtx)
					}),
					layout.Flexed(1, tableCell(th, p, name, false)),
					layout.Flexed(1, tableCell(th, p, phone, false)),
				)
			},
		)
	}
}

func resultsTable(th *material.Theme, p uiPalette, rows []types.StoreInfo) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if len(rows) == 0 {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx,
				labelMuted(th, p, "Tabel akan terisi setelah scraping selesai.").Layout,
			)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(tableHeaderBar(th, p)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ch := []layout.FlexChild{layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout)}
				for i, r := range rows {
					i := i
					r := r
					ch = append(ch, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return tableDataRow(th, p, i+1, r.Name, normalizePhoneDisplay(r.Phone))(gtx)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, ch...)
			}),
		)
	}
}
