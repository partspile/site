package ads

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/facet"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// NewAdFieldsPartial renders new-ad fields for the given category facets.
func NewAdFieldsPartial(facets []facet.Def, defaults facet.FormDefaults) g.Node {
	return AdFieldsPartial(NewFormConfig(defaults), facets)
}

// AdFieldsPartial renders create or edit ad fields for the given category facets.
func AdFieldsPartial(cfg AdFormConfig, facets []facet.Def) g.Node {
	f := adFields{cfg: cfg}
	nodes := []g.Node{
		fieldBlock("Title", f.cfg.fieldID("title"), f.titleInput()),
		fieldBlock("Description (optional)", f.cfg.fieldID("description"),
			f.descriptionFields()),
	}
	if !hasLocationFacet(facets) {
		nodes = append(nodes, fieldBlock("Location (optional)", f.cfg.fieldID("location"), f.locationInput()))
	}
	nodes = append(nodes, f.facetFieldNodes(facets)...)

	return Div(
		ID("category-fields"),
		Class("category-fields space-y-6"),
		g.Group(nodes),
	)
}

type adFields struct {
	cfg AdFormConfig
}

func (f adFields) facetInput(d facet.Def) g.Node {
	switch d.Form {
	case facet.FormMoney:
		return f.priceRow(d)
	case facet.FormSelect:
		return f.formSelect(d)
	case facet.FormRadio:
		return f.formRadio(d)
	case facet.FormDate:
		return f.formDate(d)
	case facet.FormCheckboxes:
		return f.formCheckboxes(d)
	case facet.FormFlag:
		return f.formFlag(d)
	case facet.FormLocation:
		return f.formLocation(d)
	default:
		if len(d.Units) > 0 {
			return f.intWithUnitRow(d)
		}
		return f.facetNumberInput(d)
	}
}

func (f adFields) formSelect(d facet.Def) g.Node {
	selected := f.cfg.Values.Facets[d.Key]
	opts := make([]g.Node, 0, len(d.FormOptions())+1)
	opts = append(opts, Option(Value(""), g.Text(selectPlaceholder(d))))
	for _, o := range d.FormOptions() {
		opt := Option(Value(o), g.Text(o))
		if o == selected {
			opt = Option(
				Value(o),
				g.Attr("selected", "selected"),
				g.Text(o),
			)
		}
		opts = append(opts, opt)
	}
	return Select(
		Name(d.Key),
		ID(f.cfg.fieldID(d.Key)),
		Class("w-36 "+controlClass),
		g.Group(opts),
	)
}

func selectPlaceholder(d facet.Def) string {
	return "Select a " + d.Key + "..."
}

func (f adFields) formRadio(d facet.Def) g.Node {
	selected := f.cfg.Values.Facets[d.Key]
	opts := d.FormOptions()
	nodes := make([]g.Node, len(opts))
	for i, o := range opts {
		id := fmt.Sprintf("%s-%s-%d", f.cfg.FieldPrefix, d.Key, i)
		attrs := []g.Node{
			Type("radio"),
			Name(d.Key),
			Value(o),
			ID(id),
		}
		if o == selected {
			attrs = append(attrs, g.Attr("checked", "checked"))
		}
		nodes[i] = Label(
			Class("field-option"),
			Input(attrs...),
			Span(g.Text(o)),
		)
	}
	return Div(Class("field-options"), g.Group(nodes))
}

func (f adFields) facetFieldNodes(facets []facet.Def) []g.Node {
	var nodes []g.Node
	var flagRun []facet.Def
	flushFlags := func() {
		if len(flagRun) == 0 {
			return
		}
		nodes = append(nodes, f.flagFieldGroup(flagRun))
		flagRun = nil
	}
	for _, d := range facets {
		if d.Kind == facet.Flag {
			flagRun = append(flagRun, d)
			continue
		}
		flushFlags()
		nodes = append(nodes, f.facetFieldBlock(d, f.facetInput(d)))
	}
	flushFlags()
	return nodes
}

func (f adFields) flagFieldGroup(defs []facet.Def) g.Node {
	items := make([]g.Node, len(defs))
	for i, d := range defs {
		items[i] = f.formFlag(d)
	}
	return Div(
		Class("field-group"),
		Div(Class("field-options flex-col items-start gap-2"), g.Group(items)),
	)
}

func (f adFields) formFlag(d facet.Def) g.Node {
	id := f.cfg.fieldID(d.Key)
	attrs := []g.Node{
		Type("checkbox"),
		Name(d.Key),
		Value("1"),
		ID(id),
	}
	if f.cfg.Values.FacetFlags[d.Key] {
		attrs = append(attrs, g.Attr("checked", "checked"))
	}
	return Label(
		Class("field-option"),
		Input(attrs...),
		Span(g.Text(d.Label)),
	)
}

func (f adFields) formDate(d facet.Def) g.Node {
	attrs := []g.Node{
		Type("date"),
		Name(d.Key),
		ID(f.cfg.fieldID(d.Key)),
		Class(controlClass),
	}
	if v := strings.TrimSpace(f.cfg.Values.Facets[d.Key]); v != "" {
		attrs = append(attrs, Value(v))
	}
	return Input(attrs...)
}

func (f adFields) formCheckboxes(d facet.Def) g.Node {
	selected := make(map[string]bool)
	for _, v := range f.cfg.Values.FacetMulti[d.Key] {
		selected[v] = true
	}
	opts := d.FormOptions()
	nodes := make([]g.Node, len(opts))
	for i, o := range opts {
		id := fmt.Sprintf("%s-%s-%d", f.cfg.FieldPrefix, d.Key, i)
		attrs := []g.Node{
			Type("checkbox"),
			Name(d.Key),
			Value(o),
			ID(id),
		}
		if selected[o] {
			attrs = append(attrs, g.Attr("checked", "checked"))
		}
		nodes[i] = Label(
			Class("field-option"),
			Input(attrs...),
			Span(g.Text(o)),
		)
	}
	return Div(Class("field-options"), g.Group(nodes))
}

func (f adFields) facetNumberInput(d facet.Def) g.Node {
	attrs := []g.Node{
		Type("number"),
		Name(d.Key),
		ID(f.cfg.fieldID(d.Key)),
		Class(fieldInputClass),
		g.Attr("min", "0"),
		g.Attr("step", "1"),
		g.Attr("inputmode", "numeric"),
	}
	if v := strings.TrimSpace(f.cfg.Values.Facets[d.Key]); v != "" {
		attrs = append(attrs, Value(v))
	}
	return Input(attrs...)
}

func (f adFields) intWithUnitRow(d facet.Def) g.Node {
	selected := d.FormDefaultUnit(f.cfg.Defaults)
	if u, ok := f.cfg.Values.FacetUnits[d.Key]; ok && u != "" {
		selected = u
	}
	unitName := d.Key + "_unit"
	numAttrs := []g.Node{
		Type("number"),
		Name(d.Key),
		ID(f.cfg.fieldID(d.Key)),
		Class("w-36 " + controlClass),
		g.Attr("min", "0"),
		g.Attr("step", "1"),
		g.Attr("inputmode", "numeric"),
	}
	if v := strings.TrimSpace(f.cfg.Values.Facets[d.Key]); v != "" {
		numAttrs = append(numAttrs, Value(v))
	}
	return Div(
		Class("flex flex-wrap items-center gap-2"),
		Input(numAttrs...),
		f.facetUnitSelect(unitName, selected, d.Units),
	)
}

func (f adFields) facetUnitSelect(name, selected string, units []string) g.Node {
	opts := make([]g.Node, len(units))
	for i, u := range units {
		opt := Option(Value(u), g.Text(u))
		if u == selected {
			opt = Option(Value(u), g.Attr("selected", "selected"), g.Text(u))
		}
		opts[i] = opt
	}
	return Select(
		Name(name),
		ID(f.cfg.fieldID(name)),
		Class(controlClass+" shrink-0"),
		g.Group(opts),
	)
}

func (f adFields) facetFieldBlock(d facet.Def, input g.Node) g.Node {
	label := d.Label
	if !d.Required {
		label += " (optional)"
	}
	fieldID := ""
	switch d.Form {
	case facet.FormRadio, facet.FormCheckboxes:
		// Group headings are not tied to a single control.
	default:
		fieldID = f.cfg.fieldID(d.Key)
	}
	return fieldBlock(label, fieldID, input)
}

func fieldBlock(labelText, fieldID string, input g.Node) g.Node {
	var labelNode g.Node
	if fieldID != "" {
		labelNode = Label(Class("field-label"), For(fieldID), g.Text(labelText))
	} else {
		labelNode = Span(Class("field-label"), g.Text(labelText))
	}
	return Div(
		Class("field-group"),
		labelNode,
		input,
	)
}

func (f adFields) titleInput() g.Node {
	attrs := []g.Node{
		Type("text"),
		Name("title"),
		ID(f.cfg.fieldID("title")),
		Class(fieldInputClass),
		g.Attr("maxlength", strconv.Itoa(config.MaxAdTitleLength)),
	}
	if f.cfg.Values.Title != "" {
		attrs = append(attrs, Value(f.cfg.Values.Title))
	}
	if f.cfg.Mode == AdFormCreate {
		attrs = append(attrs, g.Attr("autofocus", "autofocus"))
	}
	return Input(attrs...)
}

func (f adFields) descriptionFields() g.Node {
	return descriptionWithSuggestionsBox(f.cfg)
}

func hasLocationFacet(facets []facet.Def) bool {
	for _, d := range facets {
		if d.Kind == facet.Location {
			return true
		}
	}
	return false
}

func (f adFields) formLocation(d facet.Def) g.Node {
	return LocationInput(
		f.cfg.fieldID(d.Key),
		d.Key,
		f.cfg.Values.Facets[d.Key],
		d.LocationPlaceholder(),
	)
}

func (f adFields) locationInput() g.Node {
	return LocationInput(
		f.cfg.fieldID("location"),
		"location",
		f.cfg.Values.Location,
		"City, State or ZIP",
	)
}

func (f adFields) priceRow(d facet.Def) g.Node {
	view := f.cfg.Values.PriceRow
	if view.Currency == "" {
		view.Currency = d.FormDefaultCurrency(f.cfg.Defaults)
	}
	return AdPriceRow(f.cfg, d, f.cfg.Defaults, view)
}

// AdPriceRow renders the price facet with a "List as FREE" HTMX toggle.
func AdPriceRow(cfg AdFormConfig, d facet.Def, defaults facet.FormDefaults,
	view PriceRowView) g.Node {
	currencyCode := priceCurrencyCode(d, defaults, view.Currency)
	rowID := cfg.priceRowID()

	freeCheckbox := Label(
		Class("flex items-center gap-2 mb-2"),
		Input(priceFreeCheckboxAttrs(cfg, rowID, view.IsFree)...),
		Span(g.Text("List as FREE")),
	)

	var body g.Node
	if view.IsFree {
		body = Div(
			Class("price-free-state"),
			Input(Type("hidden"), Name(d.Key), Value("0")),
			Input(Type("hidden"), Name("price_currency"), Value(currencyCode)),
		)
	} else {
		amountAttrs := []g.Node{
			Type("number"),
			Name(d.Key),
			ID(cfg.fieldID(d.Key)),
			Class("w-36 " + controlClass),
			g.Attr("min", "0"),
			g.Attr("step", "1"),
			g.Attr("inputmode", "numeric"),
		}
		if amount := strings.TrimSpace(view.Amount); amount != "" && amount != "0" {
			amountAttrs = append(amountAttrs, Value(amount))
		}
		body = Div(
			Class("price-priced-state"),
			Div(
				Class("flex flex-wrap items-center gap-2"),
				Input(amountAttrs...),
				priceCurrencySelect(cfg, currencyCode, d.SupportedCurrencies()),
			),
		)
	}

	return Div(
		ID(rowID),
		freeCheckbox,
		body,
	)
}

func priceFreeCheckboxAttrs(cfg AdFormConfig, rowID string,
	checked bool) []g.Node {
	attrs := []g.Node{
		Type("checkbox"),
		Name("price_free"),
		Value("1"),
		ID(cfg.fieldID("price-free")),
		hx.Get(cfg.PriceFieldURL),
		hx.Target("#" + rowID),
		hx.Swap("outerHTML"),
		hx.Trigger("change"),
		hx.Include("#" + rowID),
	}
	if checked {
		attrs = append(attrs, g.Attr("checked", "checked"))
	}
	return attrs
}

func priceCurrencyCode(d facet.Def, defaults facet.FormDefaults,
	selected string) string {
	if selected != "" {
		return d.FormDefaultCurrency(facet.FormDefaults{Currency: selected})
	}
	return d.FormDefaultCurrency(defaults)
}

func priceCurrencySelect(cfg AdFormConfig, selected string,
	currencies []string) g.Node {
	opts := make([]g.Node, len(currencies))
	for i, code := range currencies {
		opt := Option(Value(code), g.Text(code))
		if code == selected {
			opt = Option(Value(code), g.Attr("selected", "selected"), g.Text(code))
		}
		opts[i] = opt
	}
	return Select(
		Name("price_currency"),
		ID(cfg.fieldID("price-currency")),
		Class("w-24 "+controlClass+" shrink-0"),
		g.Group(opts),
	)
}

// NewAdPriceRow renders the price row for HTMX swaps on the create form.
func NewAdPriceRow(d facet.Def, defaults facet.FormDefaults,
	view PriceRowView) g.Node {
	return AdPriceRow(NewFormConfig(defaults), d, defaults, view)
}
