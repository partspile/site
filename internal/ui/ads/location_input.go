package ads

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const (
	controlClass = "p-2 border border-zinc-300 dark:border-zinc-600 rounded-md " +
		"bg-white dark:bg-zinc-700 " +
		"placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
	fieldInputClass = "w-full " + controlClass
)

// LocationInput renders the shared location text field (search and new-ad forms).
func LocationInput(id, name, value, placeholder string,
	inputClass ...string) g.Node {
	class := fieldInputClass
	if len(inputClass) > 0 {
		class = inputClass[0]
	}
	attrs := []g.Node{
		Type("text"),
		Name(name),
		ID(id),
		Class(class),
		g.Attr("placeholder", placeholder),
	}
	if value != "" {
		attrs = append(attrs, Value(value))
	}
	return Input(attrs...)
}
