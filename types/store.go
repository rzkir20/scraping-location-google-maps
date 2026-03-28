package types

type StoreInfo struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	HasWebsite bool   `json:"-"`
	Website    string `json:"-"`
}
