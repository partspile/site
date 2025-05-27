package models

type VehicleData map[string]map[string]map[string][]string

type Ad struct {
	ID          int      `json:"id"`
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
}
