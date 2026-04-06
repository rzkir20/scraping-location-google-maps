package controllers

import "strings"

func getPhoneDisplay(phone string) string {
	if phone == "" {
		return "N/A"
	}
	return phone
}

func isJunkPlaceTitle(name string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	switch n {
	case "results", "search results", "hasil", "hasil penelusuran", "tempat", "places", "more results", "bersponsor":
		return true
	}
	if strings.HasPrefix(n, "hasil penelusuran") && len(n) < 45 {
		return true
	}
	return false
}
