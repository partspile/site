package ad

// SearchQuery represents a structured query for filtering ads
type SearchQuery struct {
	Make        string   `json:"make,omitempty"`
	Years       []string `json:"years,omitempty"`
	Models      []string `json:"models,omitempty"`
	EngineSizes []string `json:"engine_sizes,omitempty"`
	Category    string   `json:"category,omitempty"`
	SubCategory string   `json:"sub_category,omitempty"`
}

func (sq SearchQuery) IsEmpty() bool {
	return sq.Make == "" && len(sq.Years) == 0 && len(sq.Models) == 0 &&
		len(sq.EngineSizes) == 0 && sq.Category == "" && sq.SubCategory == ""
}
