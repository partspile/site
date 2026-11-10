package facet

import (
	"strconv"
	"testing"
	"time"

	"github.com/partspile/site/internal/currency"
)

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }

func TestIntFacetFormatting(t *testing.T) {
	mileage, ok := Get("mileage")
	if !ok {
		t.Fatal("mileage facet not registered")
	}
	v := Value{Num: intPtr(45300), Text: strPtr("mi")}
	if got := mileage.FormatCompact(v); got != "45K mi" {
		t.Errorf("FormatCompact = %q, want %q", got, "45K mi")
	}
	if got := mileage.FormatFull(v); got != "45,300 mi" {
		t.Errorf("FormatFull = %q, want %q", got, "45,300 mi")
	}

	zero := Value{Num: intPtr(0), Text: strPtr("mi")}
	if got := mileage.FormatCompact(zero); got != "" {
		t.Errorf("FormatCompact(0) = %q, want empty", got)
	}
	if got := mileage.FormatFull(zero); got != "0 mi" {
		t.Errorf("FormatFull(0) = %q, want %q", got, "0 mi")
	}

	hours, _ := Get("hours")
	hv := Value{Num: intPtr(1200)}
	if got := hours.FormatFull(hv); got != "1,200 hrs" {
		t.Errorf("hours FormatFull = %q, want %q", got, "1,200 hrs")
	}
	if got := hours.FormatCompact(Value{Num: intPtr(0)}); got != "" {
		t.Errorf("hours FormatCompact(0) = %q, want empty", got)
	}
}

func TestIntFacetValidate(t *testing.T) {
	mileage, _ := Get("mileage")
	if err := mileage.Validate(Value{Num: intPtr(100), Text: strPtr("mi")}); err != nil {
		t.Errorf("valid mileage rejected: %v", err)
	}
	if err := mileage.Validate(Value{Num: intPtr(100)}); err == nil {
		t.Error("mileage without unit should be invalid")
	}
	if err := mileage.Validate(Value{Num: intPtr(100), Text: strPtr("ft")}); err == nil {
		t.Error("unknown unit should be invalid")
	}
	if err := mileage.Validate(Value{Num: intPtr(-1), Text: strPtr("mi")}); err == nil {
		t.Error("negative mileage should be invalid")
	}

	optionalMileage := Def{Key: "mileage", Label: "Mileage", Kind: Int, Form: FormNumber, Units: []string{"mi", "km"}}
	if err := optionalMileage.Validate(Value{}); err != nil {
		t.Errorf("empty optional value should be valid: %v", err)
	}
}

func TestYearFormSelect(t *testing.T) {
	year, ok := Get("year")
	if !ok {
		t.Fatal("year facet not registered")
	}
	if year.Form != FormSelect {
		t.Fatalf("year Form = %v, want FormSelect", year.Form)
	}
	if year.Kind != Int {
		t.Fatalf("year Kind = %v, want Int", year.Kind)
	}
	if year.Filter != FilterRange {
		t.Fatalf("year Filter = %v, want FilterRange", year.Filter)
	}
	opts := year.FormOptions()
	if len(opts) == 0 {
		t.Fatal("expected generated year options")
	}
	if opts[0] != strconv.Itoa(time.Now().Year()+1) {
		t.Errorf("first option = %q, want latest year", opts[0])
	}
	min, max := year.SelectBounds()
	if min != 1900 {
		t.Errorf("SelectBounds min = %d, want 1900", min)
	}
	wantMax := time.Now().Year() + 1
	if max != wantMax {
		t.Errorf("SelectBounds max = %d, want %d", max, wantMax)
	}
	if err := year.Validate(Value{Num: intPtr(2020)}); err != nil {
		t.Errorf("valid year rejected: %v", err)
	}
	if err := year.Validate(Value{Num: intPtr(1800)}); err == nil {
		t.Error("year below range should be invalid")
	}
	if got := year.FormatFull(Value{Num: intPtr(2020)}); got != "2020" {
		t.Errorf("FormatFull = %q, want 2020", got)
	}
}

func TestDefNormalizeUnit(t *testing.T) {
	mileage, _ := Get("mileage")
	if got := mileage.NormalizeUnit("km"); got != "km" {
		t.Errorf("NormalizeUnit(km) = %q, want km", got)
	}
	if got := mileage.NormalizeUnit("ft"); got != "mi" {
		t.Errorf("NormalizeUnit(ft) = %q, want mi (first unit)", got)
	}
}

func TestMoneyFacet(t *testing.T) {
	price, ok := Get("price")
	if !ok {
		t.Fatal("price facet not registered")
	}
	v := Value{Num: intPtr(22500), Text: strPtr("USD")}
	if got, want := price.FormatFull(v), currency.Format(22500, "USD"); got != want {
		t.Errorf("price FormatFull = %q, want %q", got, want)
	}
	free := Value{Num: intPtr(0), Text: strPtr("USD")}
	if got, want := price.FormatFull(free), currency.Format(0, "USD"); got != want {
		t.Errorf("free price FormatFull = %q, want %q", got, want)
	}
	if err := price.Validate(Value{Num: intPtr(100)}); err == nil {
		t.Error("price without currency should be invalid")
	}
	if err := price.Validate(v); err != nil {
		t.Errorf("valid price rejected: %v", err)
	}
}

func TestRequiredFacet(t *testing.T) {
	required := Def{Key: "hours", Label: "Hours", Kind: Int, Required: true}
	if err := required.Validate(Value{}); err == nil {
		t.Error("required Int without value should be invalid")
	}
	if err := required.Validate(Value{Num: intPtr(10)}); err != nil {
		t.Errorf("required Int with value rejected: %v", err)
	}

	optional := Def{Key: "hours", Label: "Hours", Kind: Int}
	if err := optional.Validate(Value{}); err != nil {
		t.Errorf("optional Int without value should be valid: %v", err)
	}
}

func TestEnumFacet(t *testing.T) {
	condition, ok := Get("condition")
	if !ok {
		t.Fatal("condition facet not registered")
	}
	if condition.Form != FormRadio {
		t.Fatalf("condition Form = %v, want FormRadio", condition.Form)
	}
	if condition.Filter != FilterCheckboxes {
		t.Fatalf("condition Filter = %v, want FilterCheckboxes", condition.Filter)
	}

	d := Def{
		Key: "condition", Label: "Condition", Kind: Enum,
		Form: FormRadio, Filter: FilterCheckboxes,
		Enum: condition.Enum,
	}
	if got := d.FormatFull(Value{Text: strPtr("Used - Good")}); got != "Used - Good" {
		t.Errorf("enum FormatFull = %q, want %q", got, "Used - Good")
	}
	if err := d.Validate(Value{Text: strPtr("Used - Good")}); err != nil {
		t.Errorf("valid enum rejected: %v", err)
	}
	if err := d.Validate(Value{Text: strPtr("broken")}); err == nil {
		t.Error("invalid enum value should be rejected")
	}
	if err := d.Validate(Value{}); err != nil {
		t.Errorf("empty enum should be valid (optional): %v", err)
	}
}

func TestFormDefaultCurrency(t *testing.T) {
	price, ok := Get("price")
	if !ok {
		t.Fatal("price facet not registered")
	}
	if got := price.FormDefaultCurrency(FormDefaults{Currency: "eur"}); got != "EUR" {
		t.Errorf("FormDefaultCurrency = %q, want EUR", got)
	}
	if got := price.FormDefaultCurrency(FormDefaults{Currency: "XYZ"}); got != currency.Default {
		t.Errorf("unsupported currency should fall back to default, got %q", got)
	}

	mileage, _ := Get("mileage")
	if got := mileage.FormDefaultCurrency(FormDefaults{Currency: "USD"}); got != "" {
		t.Errorf("non-money facet should return empty currency, got %q", got)
	}
}

func TestSupportedCurrencies(t *testing.T) {
	price, _ := Get("price")
	if len(price.SupportedCurrencies()) == 0 {
		t.Error("money facet should return supported currencies")
	}
	mileage, _ := Get("mileage")
	if mileage.SupportedCurrencies() != nil {
		t.Error("non-money facet should return nil currencies")
	}
}

func TestFormDefaultUnit(t *testing.T) {
	mileage, ok := Get("mileage")
	if !ok {
		t.Fatal("mileage facet not registered")
	}
	if got := mileage.FormDefaultUnit(FormDefaults{Unit: "km"}); got != "km" {
		t.Errorf("FormDefaultUnit = %q, want km", got)
	}
	if got := mileage.FormDefaultUnit(FormDefaults{Unit: "invalid"}); got != "mi" {
		t.Errorf("invalid unit should fall back to first allowed, got %q", got)
	}

	hours, _ := Get("hours")
	if got := hours.FormDefaultUnit(FormDefaults{Unit: "km"}); got != "" {
		t.Errorf("facet without units should return empty, got %q", got)
	}
}

func TestCardLabel(t *testing.T) {
	for _, key := range []string{"mileage", "hours"} {
		d, ok := Get(key)
		if !ok {
			t.Fatalf("%s facet not registered", key)
		}
		if !d.CardLabel() {
			t.Errorf("%s should appear on listing cards", key)
		}
	}
	for _, key := range []string{"year", "condition", "price"} {
		d, ok := Get(key)
		if !ok {
			t.Fatalf("%s facet not registered", key)
		}
		if d.CardLabel() {
			t.Errorf("%s should not appear on listing cards", key)
		}
	}
	pickup, ok := Get("local_pickup")
	if !ok || pickup.CardLabel() {
		t.Error("local_pickup should not appear on listing cards")
	}
}

func TestFlagFacet(t *testing.T) {
	pickup, ok := Get("local_pickup")
	if !ok {
		t.Fatal("local_pickup facet not registered")
	}
	if pickup.Kind != Flag {
		t.Fatalf("Kind = %v, want Flag", pickup.Kind)
	}
	if pickup.Form != FormFlag {
		t.Fatalf("Form = %v, want FormFlag", pickup.Form)
	}
	if pickup.Filter != FilterFlag {
		t.Fatalf("Filter = %v, want FilterFlag", pickup.Filter)
	}
	if pickup.Required {
		t.Error("local_pickup should be optional")
	}
	if !pickup.Filterable {
		t.Error("local_pickup should be filterable")
	}
	if err := pickup.Validate(Value{}); err != nil {
		t.Errorf("empty flag should be valid: %v", err)
	}
	v := EncodeFlag(true)
	if err := pickup.Validate(v); err != nil {
		t.Errorf("valid flag rejected: %v", err)
	}
	if got := pickup.FormatFull(v); got != "Local pick-up only" {
		t.Errorf("FormatFull = %q", got)
	}
	if got := pickup.EmbeddingSnippet(v); got != "Local pick-up only" {
		t.Errorf("EmbeddingSnippet = %q", got)
	}
	meta, ok := pickup.VectorMetadataValue(v)
	if !ok || meta != 1 {
		t.Errorf("VectorMetadataValue = %v ok=%v", meta, ok)
	}
	n := 2
	if err := pickup.Validate(Value{Num: &n}); err == nil {
		t.Error("invalid flag value should be rejected")
	}
}

func TestLocationFacet(t *testing.T) {
	location, ok := Get("location")
	if !ok {
		t.Fatal("location facet not registered")
	}
	if location.Kind != Location {
		t.Fatalf("Kind = %v, want Location", location.Kind)
	}
	if location.Form != FormLocation {
		t.Fatalf("Form = %v, want FormLocation", location.Form)
	}
	if location.Required {
		t.Error("location facet should be optional")
	}
	v := Value{Text: strPtr("Portland, OR")}
	if err := location.Validate(v); err != nil {
		t.Errorf("valid location rejected: %v", err)
	}
	if got := location.FormatFull(v); got != *v.Text {
		t.Errorf("FormatFull = %q", got)
	}
	if got := location.LocationPlaceholder(); got == "" {
		t.Error("expected location placeholder")
	}
}
