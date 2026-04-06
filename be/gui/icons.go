package gui

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// Ikon vektor Material (IconVG) — tidak bergantung font emoji sistem.
var (
	iconMapsMap      *widget.Icon
	iconThemeSun     *widget.Icon
	iconThemeDark    *widget.Icon
	iconFileCSV      *widget.Icon
	iconFileJSON     *widget.Icon
	iconPlayScrape   *widget.Icon
	iconRescrape     *widget.Icon
	iconFollowGH     *widget.Icon
	iconFollowTikTok *widget.Icon
	iconFollowIG     *widget.Icon
)

func init() {
	must := func(ic *widget.Icon, err error) *widget.Icon {
		if err != nil {
			panic(err)
		}
		return ic
	}
	iconMapsMap = must(widget.NewIcon(icons.MapsMap))
	iconThemeSun = must(widget.NewIcon(icons.ImageWBSunny))
	iconThemeDark = must(widget.NewIcon(icons.ImageBrightness1))
	iconFileCSV = must(widget.NewIcon(icons.FileFileDownload))
	iconFileJSON = must(widget.NewIcon(icons.ActionCode))
	iconPlayScrape = must(widget.NewIcon(icons.AVPlayArrow))
	iconRescrape = must(widget.NewIcon(icons.NavigationRefresh))
	iconFollowGH = must(widget.NewIcon(icons.ActionCode))
	iconFollowTikTok = must(widget.NewIcon(icons.AVMusicVideo))
	iconFollowIG = must(widget.NewIcon(icons.ImagePhotoCamera))
}
