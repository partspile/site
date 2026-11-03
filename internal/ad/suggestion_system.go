package ad

import (
	"fmt"
	"sort"
	"strings"
)

// suggestionsSharedHeader is identical for every category so it can be cached
// as the leading part of the per-category system prompt (see x-grok-conv-id).
const suggestionsSharedHeader = `You help classified ad posters fill in
optional details missing from their ad copy. Return JSON only: an array of
{"label":"...","value":"..."}. No markdown, no prose. At most 12 items.

Goal: suggest choices for specs a buyer would ask about that are not already
clear from the AD COPY.

RULES:

- Do NOT suggest specs already obvious from the AD COPY
- Do NOT suggest specs already in the FORMAL FORM FIELDS (these are already
  collected by the ad poster)
- Do NOT suggest specs already in the ALREADY SELECTED list (these are already
  chosen by the ad poster)

- Focus on missing specs; infer specs from ad CATEGORY.  What specs are buyers
most interested in for this category?  What specs are most common for this
category?  Are there specs particular to this ad subject that would be reconized
by a viewer familair with the ad subject but not obvious from the AD COPY?

- When an spec has multiple plausible values, emit a separate entry for each
realistic option (same label, different values).  For example:

{"label":"transmission","value":"manual"}
{"label":"transmission","value":"automatic"}

{"label":"bedrooms","value":"3"}
{"label":"bedrooms","value":"4"}

- For binary features (present or not: navigation, non-smoking, heated seats,
pets allowed, etc.), emit ONE entry with value "yes": {"label":"heated
seats","value":"yes"}. Never emit yes/no pairs.
`

// categorySuggestionInstructions holds the category-specific guidance block,
// keyed by category name. Keep each value constant so the assembled system
// prompt stays byte-identical across requests for the same category.
var categorySuggestionInstructions = map[string]string{
	"Cars & Trucks": `Buyers of cars and trucks care about drivetrain and
usability specs, such as:
- transmission: manual | automatic
- fuel: gas | diesel | electric | hybrid
- drivetrain: FWD | RWD | AWD | 4WD
- cylinders, engine size, doors, seating

Tailor options to the specific make, model, and year.`,

	"Car & Truck Parts": `Buyers of car and truck parts care about fitment
and condition details, such as:
- fits make/model/year or generation
- part type (OEM | aftermarket)
- side (left | right | front | rear), position
- material/finish, whether the part is new, used, or rebuilt

Tailor options to the specific part described.`,

	"Motorcycles": `Buyers of motorcycles care about:
- engine size (cc), engine type (2-stroke | 4-stroke)
- cylinders, cooling (air | liquid)
- type (sport | cruiser | touring | dirt | adventure)

Tailor options to the specific make, model, and year.`,

	"Motorcycle Parts": `Buyers of motorcycle parts care about fitment and
condition, such as:
- fits make/model/year
- part type (OEM | aftermarket)
- position/side, material/finish

Tailor options to the specific part described.`,

	"Bicycles": `Buyers of bicycles care about:
- type (road | mountain | gravel | hybrid | BMX | e-bike)
- frame size, frame material (aluminum | carbon | steel | titanium)
- wheel size, drivetrain/speeds, brake type (disc | rim)
- suspension (none | front | full)

Tailor options to the specific make and year.`,

	"Bicycle Parts": `Buyers of bicycle parts care about compatibility and
condition, such as:
- component group/standard, wheel or tire size
- material, mounting standard
- new or used

Tailor options to the specific part described.`,

	"Agricultural Equipment": `Buyers of agricultural equipment care about:
- equipment type, drive (2WD | 4WD)
- horsepower (PTO/engine), fuel (diesel | gas)
- cab vs open station

Tailor options to the specific make and year.`,

	"Agricultural Equipment Parts": `Buyers of agricultural equipment parts
care about fitment and condition, such as:
- fits make/model/series
- part type (OEM | aftermarket)

Tailor options to the specific part described.`,
}

// suggestionsSystemPrompt builds the constant-per-category system prompt:
// shared header + formal fields + category guidance.
func suggestionsSystemPrompt(categoryName string,
	facets map[string]string) string {
	systemPrompt := suggestionsSharedHeader
	systemPrompt += "\nAd CATEGORY: " + categoryName
	systemPrompt += "\n" + categorySuggestionInstructions[categoryName]
	systemPrompt += "\nFORMAL FORM FIELDS already collected (never suggest these): " +
		formalFacetLabelList(facets)
	return systemPrompt
}

func formalFacetLabelList(facets map[string]string) string {
	if len(facets) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(facets))
	for k := range facets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labels := make([]string, len(keys))
	for i, k := range keys {
		labels[i] = facets[k]
	}
	return strings.Join(labels, ", ")
}

func suggestionsConvID(categoryID int) string {
	return fmt.Sprintf("ad-suggestions-cat-%d", categoryID)
}
