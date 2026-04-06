package types

// LiveCard menyimpan info kartu yang sedang diproses di Google Maps.
type LiveCard struct {
	Name          string `json:"name"`
	Rating        string `json:"rating,omitempty"`
	Category      string `json:"category,omitempty"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty"`
	OpeningStatus string `json:"openingStatus,omitempty"`
}
