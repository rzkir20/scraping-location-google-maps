package types

type StoreInfo struct {
	Name       string `json:"name"`
	Rating     string `json:"rating,omitempty"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	HasWebsite bool   `json:"-"`
	Website    string `json:"-"`
}
