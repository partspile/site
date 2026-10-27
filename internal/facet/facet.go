// Package facet defines the category-specific ad attributes (price, mileage,
// hours, year, condition, ...) in one typed registry. Each facet's kind,
// unit handling, allowed values, validation, and display formatting live in a
// single Def so that adding a facet is a one-line registry entry rather than a
// schema/SQL/UI change.
package facet

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/location"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type Kind int

const (
	Int       Kind = iota // numeric, range-filterable
	Money                 // amount + currency (Value.Text), FREE when amount == 0
	Enum                  // fixed string set, exact-match filter
	Date                  // ISO date YYYY-MM-DD in Value.Text
	MultiEnum             // multiple enum selections, JSON array in Value.Text
	Location              // user location text; resolves to ads.location_id
	Flag                  // optional opt-in; Num == 1 when set
)

const dateLayout = "2006-01-02"
const flagSetValue = 1

// FormWidget controls the new-ad form control for a facet.
type FormWidget int

const (
	FormNumber     FormWidget = iota // plain number input (+ unit select when Units set)
	FormSelect                       // <select>; options from Enum (Kind Enum) or SelectMin..SelectMax (Kind Int)
	FormRadio                        // exclusive choices for Kind Enum
	FormMoney                        // amount + currency + FREE
	FormDate                         // date input (Kind Date)
	FormCheckboxes                   // multi-select checkboxes (Kind MultiEnum)
	FormLocation                     // location text input (Kind Location)
	FormFlag                         // single opt-in checkbox (Kind Flag)
)

// FilterWidget controls the search filter panel for a facet.
type FilterWidget int

const (
	FilterRange      FilterWidget = iota // min/max inputs
	FilterExact                          // single-select (Enum dropdown)
	FilterCheckboxes                     // multi-select (Enum checkboxes)
	FilterFlag                           // single opt-in checkbox (Flag)
)

// FormDefaults holds per-request values for new-ad form widgets.
type FormDefaults struct {
	Currency string // default selected currency for money facets
	Unit     string // preferred unit for facets with Units (e.g. mi/km)
}

// Def is the definition of a single facet.
type Def struct {
	Key        string
	Label      string
	Kind       Kind
	Form       FormWidget
	Filter     FilterWidget
	Units      []string // Int-only: allowed units stored in Value.Text; empty = no unit
	Suffix     string   // Int-only: constant suffix when Units is empty (e.g. "hrs")
	Compact    bool     // Int-only: render large values as e.g. "45K" on cards
	Enum       []string // enum options for FormSelect and FormRadio
	SelectMin  int      // FormSelect lower bound when Kind == Int (inclusive)
	SelectMax  int      // FormSelect upper bound when Kind == Int; 0 = current calendar year + 1
	Filterable bool     // show in the search filter panel
	Required   bool     // required on the new-ad form
}

// CardLabel reports whether this facet appears after the title on listing cards.
func (d Def) CardLabel() bool {
	return d.Key == "mileage" || d.Key == "hours" || d.Key == "sale_start_date"
}

// LocationPlaceholder returns the form placeholder for a Location facet.
func (d Def) LocationPlaceholder() string {
	if d.Key == "address" {
		return "Street address, City, State or ZIP"
	}
	return "City, State or ZIP"
}

// ValidFilterValue reports whether val is an allowed search filter option.
func (d Def) ValidFilterValue(val string) bool {
	for _, e := range d.Enum {
		if e == val {
			return true
		}
	}
	return false
}

// ValidUnit reports whether unit is allowed for this facet.
func (d Def) ValidUnit(unit string) bool {
	unit = strings.TrimSpace(unit)
	for _, u := range d.Units {
		if u == unit {
			return true
		}
	}
	return false
}

// SupportedCurrencies returns currency dropdown options for money facets.
func (d Def) SupportedCurrencies() []string {
	if d.Form != FormMoney {
		return nil
	}
	return currency.Supported
}

// FormDefaultCurrency returns the selected currency for a money facet.
func (d Def) FormDefaultCurrency(defaults FormDefaults) string {
	if d.Form != FormMoney {
		return ""
	}
	code := currency.Normalize(defaults.Currency)
	if currency.IsSupported(code) {
		return code
	}
	return currency.Default
}

// FormDefaultUnit returns the selected unit for a facet with Units.
func (d Def) FormDefaultUnit(defaults FormDefaults) string {
	if len(d.Units) == 0 {
		return ""
	}
	return d.NormalizeUnit(defaults.Unit)
}

// NormalizeUnit returns unit if allowed, otherwise the first allowed unit.
func (d Def) NormalizeUnit(unit string) string {
	unit = strings.TrimSpace(unit)
	if d.ValidUnit(unit) {
		return unit
	}
	if len(d.Units) > 0 {
		return d.Units[0]
	}
	return unit
}

// SelectBounds returns inclusive min/max for FormSelect on Kind Int.
func (d Def) SelectBounds() (min, max int) {
	min = d.SelectMin
	if d.SelectMax != 0 {
		max = d.SelectMax
	} else {
		max = time.Now().Year() + 1
	}
	if max < min {
		max = min
	}
	return min, max
}

// FormOptions returns dropdown or checkbox choices for form widgets.
func (d Def) FormOptions() []string {
	switch {
	case d.Kind == Enum || d.Kind == MultiEnum:
		return d.Enum
	case d.Form == FormSelect && d.Kind == Int:
		min, max := d.SelectBounds()
		opts := make([]string, 0, max-min+1)
		for v := max; v >= min; v-- {
			opts = append(opts, strconv.Itoa(v))
		}
		return opts
	default:
		return nil
	}
}

// Value is a stored facet value for one ad. Num holds Int/Money amounts; Text
// holds currency (Money), unit (Int with Units), selection (Enum/Date), or a
// JSON array (MultiEnum). Values holds decoded MultiEnum selections in memory.
type Value struct {
	Num    *int
	Text   *string
	Values []string
}

// Filter is a search constraint. Min/Max apply to Int/Money; TextMin/TextMax to
// Date; Value to a single Enum; Values to multiple Enum selections (OR within
// the facet).
type Filter struct {
	Min     *int     `json:"min,omitempty"`
	Max     *int     `json:"max,omitempty"`
	TextMin *string  `json:"text_min,omitempty"`
	TextMax *string  `json:"text_max,omitempty"`
	Value   *string  `json:"value,omitempty"`
	Values  []string `json:"values,omitempty"`
}

func (v Value) Present() bool {
	return v.Num != nil || v.Text != nil || len(v.Values) > 0
}

func (f Filter) Active() bool {
	return f.Min != nil || f.Max != nil || f.TextMin != nil ||
		f.TextMax != nil || f.Value != nil || len(f.Values) > 0
}

// EncodeFlag builds a Flag value when checked.
func EncodeFlag(checked bool) Value {
	if !checked {
		return Value{}
	}
	n := flagSetValue
	return Value{Num: &n}
}

// FlagSet reports whether a Flag facet value is checked.
func (v Value) FlagSet() bool {
	return v.Num != nil && *v.Num != 0
}

// ParseDateValue parses an ISO date string into a Date facet value.
func ParseDateValue(raw string) (Value, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Value{}, fmt.Errorf("date is required")
	}
	if _, err := time.Parse(dateLayout, raw); err != nil {
		return Value{}, fmt.Errorf("invalid date")
	}
	return Value{Text: &raw}, nil
}

// DateString returns the ISO date from a Date facet value.
func (v Value) DateString() string {
	if v.Text == nil {
		return ""
	}
	return *v.Text
}

// EncodeMultiEnum builds a MultiEnum value from selected options.
func EncodeMultiEnum(selected []string) Value {
	if len(selected) == 0 {
		return Value{}
	}
	dup := append([]string(nil), selected...)
	sort.Strings(dup)
	data, err := json.Marshal(dup)
	if err != nil {
		return Value{}
	}
	s := string(data)
	return Value{Text: &s, Values: dup}
}

// MultiEnumValues decodes MultiEnum selections from memory or stored JSON.
func (v Value) MultiEnumValues() []string {
	if len(v.Values) > 0 {
		return append([]string(nil), v.Values...)
	}
	if v.Text == nil || *v.Text == "" {
		return nil
	}
	var vals []string
	if err := json.Unmarshal([]byte(*v.Text), &vals); err != nil {
		return nil
	}
	return vals
}

// FormatDateRange renders a compact date or date range for listing cards.
func FormatDateRange(start, end string) string {
	if start == "" {
		return ""
	}
	if end == "" || end == start {
		return formatDateDisplay(start)
	}
	startT, err1 := time.Parse(dateLayout, start)
	endT, err2 := time.Parse(dateLayout, end)
	if err1 != nil || err2 != nil {
		return formatDateDisplay(start)
	}
	if startT.Month() == endT.Month() && startT.Year() == endT.Year() {
		return fmt.Sprintf("%s %d–%d, %d",
			startT.Format("Jan"), startT.Day(), endT.Day(), startT.Year())
	}
	return formatDateDisplay(start) + "–" + formatDateDisplay(end)
}

// SaleDateDetailDisplays returns bullet labels for sale dates on the ad detail page.
func SaleDateDetailDisplays(start, end string) []string {
	if s := FormatDateRange(start, end); s != "" {
		return []string{"Sale Date: " + s}
	}
	return nil
}

func formatDateDisplay(iso string) string {
	t, err := time.Parse(dateLayout, iso)
	if err != nil {
		return iso
	}
	return t.Format("Jan 2, 2006")
}

// ResolveFilterForSearch converts stored filter presets into query constraints.
func ResolveFilterForSearch(key string, f Filter, now time.Time) Filter {
	if key != "sale_start_date" || f.Value == nil {
		return f
	}
	min, max, ok := SaleWeekRange(*f.Value, now)
	if !ok {
		return f
	}
	minS := min.Format(dateLayout)
	maxS := max.Format(dateLayout)
	return Filter{TextMin: &minS, TextMax: &maxS}
}

// SaleWeekRange maps a when-filter preset to an inclusive sale_start_date range.
func SaleWeekRange(preset string, now time.Time) (min, max time.Time, ok bool) {
	loc := now.Location()
	today := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc,
	)
	weekStart := weekStartSunday(today)

	switch preset {
	case "This week":
		return weekStart, weekStart.AddDate(0, 0, 6), true
	case "Next week":
		start := weekStart.AddDate(0, 0, 7)
		return start, start.AddDate(0, 0, 6), true
	case "This weekend":
		sat, sun := thisWeekend(today)
		return sat, sun, true
	case "Next weekend":
		sat, sun := thisWeekend(today)
		return sat.AddDate(0, 0, 7), sun.AddDate(0, 0, 7), true
	case "Later":
		start := weekStart.AddDate(0, 0, 14)
		far := time.Date(9999, 12, 31, 0, 0, 0, 0, loc)
		return start, far, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

func weekStartSunday(t time.Time) time.Time {
	return t.AddDate(0, 0, -int(t.Weekday()))
}

func thisWeekend(today time.Time) (sat, sun time.Time) {
	switch today.Weekday() {
	case time.Sunday:
		return today.AddDate(0, 0, -1), today
	case time.Saturday:
		return today, today.AddDate(0, 0, 1)
	default:
		daysUntilSat := int(time.Saturday - today.Weekday())
		sat = today.AddDate(0, 0, daysUntilSat)
		return sat, sat.AddDate(0, 0, 1)
	}
}

var defs = []Def{
	{
		Key:        "price",
		Label:      "Price",
		Kind:       Money,
		Form:       FormMoney,
		Filter:     FilterRange,
		Filterable: true,
		Required:   true,
	},
	{
		Key:        "mileage",
		Label:      "Mileage",
		Kind:       Int,
		Form:       FormNumber,
		Filter:     FilterRange,
		Units:      []string{location.UnitMiles, location.UnitKm},
		Compact:    true,
		Filterable: true,
		Required:   true,
	},
	{
		Key:        "hours",
		Label:      "Hours",
		Kind:       Int,
		Form:       FormNumber,
		Filter:     FilterRange,
		Suffix:     "hrs",
		Filterable: true,
		Required:   true,
	},
	{
		Key:        "year",
		Label:      "Year",
		Kind:       Int,
		Form:       FormSelect,
		Filter:     FilterRange,
		SelectMin:  1900,
		Filterable: true,
		Required:   true,
	},
	{
		Key:    "condition",
		Label:  "Condition",
		Kind:   Enum,
		Form:   FormRadio,
		Filter: FilterCheckboxes,
		Enum: []string{
			"New",
			"Used - Like new",
			"Used - Good",
			"Used - Fair",
			"Used - Poor",
		},
		Filterable: true,
		Required:   false,
	},
	{
		Key:    "title_status",
		Label:  "Title Status",
		Kind:   Enum,
		Form:   FormRadio,
		Filter: FilterCheckboxes,
		Enum: []string{
			"Clean",
			"Salvage",
			"Rebuilt",
			"Parts Only",
		},
		Filterable: true,
		Required:   false,
	},
	{
		Key:    "sale_type",
		Label:  "Sale Type",
		Kind:   Enum,
		Form:   FormRadio,
		Filter: FilterCheckboxes,
		Enum: []string{
			"Garage Sale",
			"Estate Sale",
			"Moving Sale",
			"Yard Sale",
			"Multi-Family Sale",
		},
		Filterable: true,
		Required:   true,
	},
	{
		Key:    "sale_start_date",
		Label:  "When",
		Kind:   Date,
		Form:   FormDate,
		Filter: FilterExact,
		Enum: []string{
			"This week",
			"Next week",
			"This weekend",
			"Next weekend",
			"Later",
		},
		Filterable: true,
		Required:   true,
	},
	{
		Key:        "sale_end_date",
		Label:      "Sale End Date",
		Kind:       Date,
		Form:       FormDate,
		Filter:     FilterRange,
		Filterable: false,
		Required:   true,
	},
	{
		Key:    "pricing_style",
		Label:  "Pricing Style",
		Kind:   MultiEnum,
		Form:   FormCheckboxes,
		Filter: FilterCheckboxes,
		Enum: []string{
			"Everything Priced",
			"Negotiable / Make Offer",
			"Everything Must Go",
			"Free Items Included",
		},
		Filterable: true,
		Required:   false,
	},
	{
		Key:        "local_pickup",
		Label:      "Local pick-up only",
		Kind:       Flag,
		Form:       FormFlag,
		Filter:     FilterFlag,
		Filterable: true,
		Required:   false,
	},
	{
		Key:        "location",
		Label:      "Location",
		Kind:       Location,
		Form:       FormLocation,
		Filterable: false,
		Required:   false,
	},
	{
		Key:        "address",
		Label:      "Address",
		Kind:       Location,
		Form:       FormLocation,
		Filterable: false,
		Required:   true,
	},
}

var byKey = func() map[string]Def {
	m := make(map[string]Def, len(defs))
	for _, d := range defs {
		m[d.Key] = d
	}
	return m
}()

// Get returns the definition for a facet key.
func Get(key string) (Def, bool) {
	d, ok := byKey[key]
	return d, ok
}

// All returns every registered facet definition.
func All() []Def {
	return defs
}

// Validate checks a value against the definition. Optional facets may be absent;
// Required facets must be present and pass kind-specific rules.
func (d Def) Validate(v Value) error {
	switch d.Kind {
	case Int:
		if v.Num == nil {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
		if *v.Num < 0 {
			return fmt.Errorf("%s must be zero or greater", d.Label)
		}
		if len(d.Units) > 0 && (v.Text == nil || !d.ValidUnit(*v.Text)) {
			return fmt.Errorf("%s requires a valid unit", d.Label)
		}
		if d.Form == FormSelect && d.Kind == Int {
			min, max := d.SelectBounds()
			if *v.Num < min || *v.Num > max {
				return fmt.Errorf("%s must be between %d and %d", d.Label, min, max)
			}
		}
	case Money:
		if v.Num == nil {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
		if *v.Num < 0 {
			return fmt.Errorf("%s must be zero or greater", d.Label)
		}
		if v.Text == nil || !currency.IsSupported(*v.Text) {
			return fmt.Errorf("%s requires a valid currency", d.Label)
		}
	case Enum:
		if v.Text == nil || strings.TrimSpace(*v.Text) == "" {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
		for _, e := range d.Enum {
			if e == *v.Text {
				return nil
			}
		}
		return fmt.Errorf("invalid %s", d.Label)
	case Date:
		if v.Text == nil || strings.TrimSpace(*v.Text) == "" {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
		if _, err := time.Parse(dateLayout, *v.Text); err != nil {
			return fmt.Errorf("%s must be a valid date", d.Label)
		}
	case MultiEnum:
		vals := v.MultiEnumValues()
		if len(vals) == 0 {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
		for _, val := range vals {
			found := false
			for _, e := range d.Enum {
				if e == val {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid %s", d.Label)
			}
		}
	case Location:
		if v.Text == nil || strings.TrimSpace(*v.Text) == "" {
			if d.Required {
				return fmt.Errorf("%s is required", d.Label)
			}
			return nil
		}
	case Flag:
		if v.Num == nil {
			return nil
		}
		if *v.Num != 0 && *v.Num != flagSetValue {
			return fmt.Errorf("invalid %s", d.Label)
		}
	}
	return nil
}

// FormatCompact renders a value for dense contexts (listing cards).
func (d Def) FormatCompact(v Value) string {
	return d.format(v, d.Compact)
}

// FormatFull renders a value with full precision (ad detail page).
func (d Def) FormatFull(v Value) string {
	return d.format(v, false)
}

func (d Def) format(v Value, compact bool) string {
	switch d.Kind {
	case Money:
		if v.Num == nil {
			return ""
		}
		code := ""
		if v.Text != nil {
			code = *v.Text
		}
		return currency.Format(*v.Num, code)
	case Int:
		if v.Num == nil {
			return ""
		}
		var s string
		if d.Form == FormSelect && d.Kind == Int {
			s = strconv.Itoa(*v.Num)
		} else {
			s = formatCount(*v.Num, compact)
		}
		if len(d.Units) > 0 && v.Text != nil {
			return s + " " + *v.Text
		}
		if d.Suffix != "" {
			return s + " " + d.Suffix
		}
		return s
	case Enum:
		if v.Text == nil {
			return ""
		}
		return *v.Text
	case Date:
		if v.Text == nil {
			return ""
		}
		return formatDateDisplay(*v.Text)
	case MultiEnum:
		vals := v.MultiEnumValues()
		if len(vals) == 0 {
			return ""
		}
		return strings.Join(vals, ", ")
	case Location:
		if v.Text == nil {
			return ""
		}
		return *v.Text
	case Flag:
		if !v.FlagSet() {
			return ""
		}
		return d.Label
	}
	return ""
}

func formatCount(n int, compact bool) string {
	if compact && n >= 1000 {
		return strconv.Itoa(n/1000) + "K"
	}
	p := message.NewPrinter(language.English)
	return p.Sprint(number.Decimal(int64(n), number.Scale(0)))
}

// EmbeddingSnippet returns a prompt line for vector encoding.
func (d Def) EmbeddingSnippet(v Value) string {
	if d.Kind == Flag {
		if !v.FlagSet() {
			return ""
		}
		return d.Label
	}
	s := d.FormatFull(v)
	if s == "" {
		return ""
	}
	return d.Label + ": " + s
}

// VectorMetadataValue returns the JSONB value for search filters.
func (d Def) VectorMetadataValue(v Value) (any, bool) {
	if !v.Present() {
		return nil, false
	}
	switch d.Kind {
	case MultiEnum:
		vals := v.MultiEnumValues()
		if len(vals) == 0 {
			return nil, false
		}
		return vals, true
	case Money, Int:
		if v.Num == nil {
			return nil, false
		}
		return *v.Num, true
	case Enum, Date, Location:
		if v.Text == nil {
			return nil, false
		}
		return *v.Text, true
	case Flag:
		if !v.FlagSet() {
			return nil, false
		}
		return flagSetValue, true
	default:
		return nil, false
	}
}
