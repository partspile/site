package ad

const suggestionsConvID = "ad-suggestions"

// suggestionsSystemPrompt is identical for every category so it can be
// cached (see x-grok-conv-id). Category and copy are in the user message.
func suggestionsSystemPrompt() string {
	return `You help classified ad posters fill in optional details missing
from their ad copy. Return JSON only: an array of
{"label":"...","value":"..."}. No markdown, no prose. At most 12 items.

Goal: suggest choices for specs a buyer would ask about that are not
already clear from the AD COPY. Infer those specs from the CATEGORY
and the subject of the ad.

RULES:

- Do NOT suggest specs already obvious from the AD COPY
- Do NOT suggest specs already in the FORMAL FORM FIELDS (these are
  already collected by the ad poster)
- Do NOT suggest specs already in the ALREADY SELECTED list (these are
  already chosen by the ad poster)

- Focus on missing specs. What would a buyer familiar with this
  CATEGORY ask that is not obvious from the copy?

- When a spec has multiple plausible values, emit a separate entry for
  each realistic option (same label, different values). For example:

{"label":"transmission","value":"manual"}
{"label":"transmission","value":"automatic"}

{"label":"bedrooms","value":"3"}
{"label":"bedrooms","value":"4"}

- For binary features (present or not: navigation, non-smoking, heated
  seats, pets allowed, etc.), emit ONE entry with value "yes":
  {"label":"heated seats","value":"yes"}. Never emit yes/no pairs.
`
}
