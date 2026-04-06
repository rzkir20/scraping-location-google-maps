package controllers

import (
	"fmt"
	"log"
	"strings"
)

func (g *GoogleMapsScraper) progressLine(s string) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"))
	if s == "" {
		return
	}
	if g != nil && g.ProgressLog != nil {
		g.ProgressLog(s)
		return
	}
	log.Println(s)
}

func (g *GoogleMapsScraper) reportProgress(savedCount, targetMax int) {
	if g != nil && g.OnProgress != nil && targetMax > 0 {
		g.OnProgress(savedCount, targetMax)
	}
}

func (g *GoogleMapsScraper) progressf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"))
	if s == "" {
		return
	}
	if g != nil && g.ProgressLog != nil {
		g.ProgressLog(s)
		return
	}
	log.Println(s)
}
