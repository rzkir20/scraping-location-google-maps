package main

import (
	"image/color"

	"gioui.org/widget/material"
)

// uiPalette warna UI; dipakai untuk tema terang/gelap.
type uiPalette struct {
	PageBG    color.NRGBA
	Card      color.NRGBA
	InputFill color.NRGBA
	LogCard   color.NRGBA
	EditorBg  color.NRGBA
	Title     color.NRGBA
	Body      color.NRGBA
	Muted     color.NRGBA
	Accent    color.NRGBA
	AccentFg  color.NRGBA
	TblHead   color.NRGBA
	TblRow    color.NRGBA
	TblRowAlt color.NRGBA
	StickyBar color.NRGBA
	StickySep color.NRGBA
}

func paletteLight() uiPalette {
	return uiPalette{
		PageBG:    color.NRGBA{R: 241, G: 245, B: 249, A: 255},
		Card:      color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		InputFill: color.NRGBA{R: 248, G: 250, B: 252, A: 255},
		LogCard:   color.NRGBA{R: 235, G: 241, B: 248, A: 255},
		EditorBg:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		Title:     color.NRGBA{R: 15, G: 23, B: 42, A: 255},
		Body:      color.NRGBA{R: 30, G: 41, B: 59, A: 255},
		Muted:     color.NRGBA{R: 100, G: 116, B: 139, A: 255},
		Accent:    color.NRGBA{R: 13, G: 148, B: 136, A: 255},
		AccentFg:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		TblHead:   color.NRGBA{R: 226, G: 232, B: 240, A: 255},
		TblRow:    color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		TblRowAlt: color.NRGBA{R: 248, G: 250, B: 252, A: 255},
		StickyBar: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		StickySep: color.NRGBA{R: 226, G: 232, B: 240, A: 255},
	}
}

func paletteDark() uiPalette {
	return uiPalette{
		PageBG:    color.NRGBA{R: 15, G: 23, B: 42, A: 255},
		Card:      color.NRGBA{R: 30, G: 41, B: 59, A: 255},
		InputFill: color.NRGBA{R: 51, G: 65, B: 85, A: 255},
		LogCard:   color.NRGBA{R: 30, G: 41, B: 59, A: 255},
		EditorBg:  color.NRGBA{R: 15, G: 23, B: 42, A: 255},
		Title:     color.NRGBA{R: 241, G: 245, B: 249, A: 255},
		Body:      color.NRGBA{R: 226, G: 232, B: 240, A: 255},
		Muted:     color.NRGBA{R: 148, G: 163, B: 184, A: 255},
		Accent:    color.NRGBA{R: 45, G: 212, B: 191, A: 255},
		AccentFg:  color.NRGBA{R: 15, G: 23, B: 42, A: 255},
		TblHead:   color.NRGBA{R: 51, G: 65, B: 85, A: 255},
		TblRow:    color.NRGBA{R: 30, G: 41, B: 59, A: 255},
		TblRowAlt: color.NRGBA{R: 41, G: 53, B: 69, A: 255},
		StickyBar: color.NRGBA{R: 30, G: 41, B: 59, A: 255},
		StickySep: color.NRGBA{R: 51, G: 65, B: 85, A: 255},
	}
}

func modernTheme(p uiPalette) *material.Theme {
	th := material.NewTheme()
	th.Palette = material.Palette{
		Bg:         p.Card,
		Fg:         p.Body,
		ContrastBg: p.Accent,
		ContrastFg: p.AccentFg,
	}
	return th
}
