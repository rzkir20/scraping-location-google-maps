package types

type StoreInfo struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	HasWebsite bool   `json:"-"`
	Website    string `json:"-"`
}
