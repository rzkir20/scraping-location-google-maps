package gui

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
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
			cs := gtx.Constraints
			if cs.Max.X > 0 && cs.Max.Y > 0 {
				defer clip.Rect(image.Rectangle{Max: cs.Max}).Push(gtx.Ops).Pop()
			}
			l := material.Body2(th, txt)
			l.Color = p.Body
			l.WrapPolicy = text.WrapWords
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
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(40)
						gtx.Constraints.Max.X = gtx.Dp(52)
						return tableCell(th, p, "#", true)(gtx)
					}),
					layout.Flexed(1, tableCell(th, p, "Nama usaha", true)),
					layout.Flexed(1, tableCell(th, p, "Telepon", true)),
					layout.Flexed(1, tableCell(th, p, "Alamat", true)),
				)
			},
		)
	}
}

func tableDataRow(th *material.Theme, p uiPalette, idx int, name, phone, address string) layout.Widget {
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
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(40)
						gtx.Constraints.Max.X = gtx.Dp(52)
						return tableCell(th, p, fmt.Sprintf("%d", idx), false)(gtx)
					}),
					layout.Flexed(1, tableCell(th, p, name, false)),
					layout.Flexed(1, tableCell(th, p, phone, false)),
					layout.Flexed(1, tableCell(th, p, address, false)),
				)
			},
		)
	}
}

// tableMinWidthDp lebar minimum konten tabel (lebih lebar dari area kartu default ~648dp)
// supaya ada ruang yang digulir secara horizontal; jika tidak, contentW == viewport dan List tidak pernah scroll.
const tableMinWidthDp = 720

// resultsTable memakai scroll horizontal: satu layout.List (sumbu X) berisi tabel penuh.
// Viewport dibatasi ke lebar terhingga (bukan Max.X ~1e6) agar List menghitung scroll dengan benar.
func resultsTable(th *material.Theme, p uiPalette, rows []types.StoreInfo, scrollH *widget.List) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if len(rows) == 0 {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx,
				labelMuted(th, p, "Tabel akan terisi setelah scraping selesai.").Layout,
			)
		}

		vw := gtx.Constraints.Max.X
		if vw <= 0 || vw > 100000 {
			vw = gtx.Dp(680)
		}
		gtx.Constraints.Max.X = vw
		if gtx.Constraints.Min.X > vw {
			gtx.Constraints.Min.X = vw
		}

		minW := gtx.Dp(tableMinWidthDp)
		contentW := vw
		if contentW < minW {
			contentW = minW
		}

		scrollH.Axis = layout.Horizontal
		// Roda vertikal di area tabel juga menggeser konten horizontal (berguna saat bersarang dengan scroll halaman).
		scrollH.ScrollAnyAxis = true

		return material.List(th, scrollH).Layout(gtx, 1, func(gtx layout.Context, i int) layout.Dimensions {
			gtx.Constraints.Min.X = contentW
			gtx.Constraints.Max.X = contentW

			ch := []layout.FlexChild{
				layout.Rigid(tableHeaderBar(th, p)),
				layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
			}
			for i, r := range rows {
				i := i
				r := r
				addr := strings.TrimSpace(r.Address)
				if addr == "" {
					addr = "—"
				}
				ch = append(ch, layout.Rigid(tableDataRow(th, p, i+1, r.Name, normalizePhoneDisplay(r.Phone), addr)))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, ch...)
		})
	}
}
