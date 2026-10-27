package ads

import (
	"fmt"

	"github.com/partspile/site/internal/facet"
)

// PriceRowView holds state for the price row (initial render and HTMX swaps).
type PriceRowView struct {
	IsFree   bool
	Amount   string
	Currency string
}

// SearchFilters holds search filter panel state passed from handlers.
type SearchFilters struct {
	Facets          map[string]facet.Filter
	Location        string
	LocationDisplay string
	Within          int
	WithinUnit      string
	WithinOptions   []int
}

// SuggestionOption holds one LLM suggestion pill on the new-ad form.
type SuggestionOption struct {
	Label    string
	Value    string
	Selected bool
}

type AdFormMode int

const (
	AdFormCreate AdFormMode = iota
	AdFormEdit
)

// AdFormValues holds pre-filled field values for edit mode.
type AdFormValues struct {
	Title               string
	OriginalDescription string
	ImageCount          int
	Location            string
	Facets              map[string]string
	FacetMulti          map[string][]string
	FacetFlags          map[string]bool
	FacetUnits          map[string]string
	PriceRow            PriceRowView
	Suggestions         []SuggestionOption
}

// AdFormConfig configures shared create/edit ad form rendering.
type AdFormConfig struct {
	Mode           AdFormMode
	AdID           int
	FormID         string
	FieldPrefix    string
	PostURL        string
	PriceFieldURL  string
	SuggestionsURL string
	SubmitLabel    string
	Values         AdFormValues
	Defaults       facet.FormDefaults
}

const (
	createFormID      = "new-ad-form"
	createFieldPrefix = "new-ad"
	priceFieldURL     = "/auth/ad/new/price-field"
	adPriceRowID      = "ad-price-row"
)

// NewFormConfig returns config for the create-ad form.
func NewFormConfig(defaults facet.FormDefaults) AdFormConfig {
	return NewFormConfigWithValues(defaults, AdFormValues{})
}

// NewFormConfigWithValues returns create-ad config with prefilled values
// (e.g. copy-from-ad). Images are not prefilled.
func NewFormConfigWithValues(defaults facet.FormDefaults,
	values AdFormValues) AdFormConfig {
	if values.Facets == nil {
		values.Facets = make(map[string]string)
	}
	if values.FacetUnits == nil {
		values.FacetUnits = make(map[string]string)
	}
	if values.FacetMulti == nil {
		values.FacetMulti = make(map[string][]string)
	}
	if values.FacetFlags == nil {
		values.FacetFlags = make(map[string]bool)
	}
	values.ImageCount = 0
	return AdFormConfig{
		Mode:           AdFormCreate,
		FormID:         createFormID,
		FieldPrefix:    createFieldPrefix,
		PostURL:        "/auth/ad/new",
		PriceFieldURL:  priceFieldURL,
		SuggestionsURL: "/auth/ad/new/suggestions",
		SubmitLabel:    "Submit",
		Values:         values,
		Defaults:       defaults,
	}
}

// EditFormConfig returns config for the edit-ad form.
func EditFormConfig(adID int, values AdFormValues,
	defaults facet.FormDefaults) AdFormConfig {
	if values.Facets == nil {
		values.Facets = make(map[string]string)
	}
	if values.FacetUnits == nil {
		values.FacetUnits = make(map[string]string)
	}
	if values.FacetMulti == nil {
		values.FacetMulti = make(map[string][]string)
	}
	if values.FacetFlags == nil {
		values.FacetFlags = make(map[string]bool)
	}
	return AdFormConfig{
		Mode:           AdFormEdit,
		AdID:           adID,
		FormID:         "edit-ad-form",
		FieldPrefix:    "edit-ad",
		PostURL:        fmt.Sprintf("/auth/ad/%d/edit", adID),
		PriceFieldURL:  priceFieldURL,
		SuggestionsURL: fmt.Sprintf("/auth/ad/%d/edit/suggestions", adID),
		SubmitLabel:    "Update",
		Values:         values,
		Defaults:       defaults,
	}
}

func (c AdFormConfig) fieldID(name string) string {
	return c.FieldPrefix + "-" + name
}

func (c AdFormConfig) priceRowID() string {
	return adPriceRowID
}
