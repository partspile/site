package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)

	fmt.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	_ = Page(
		"Parts Pile - Electronic Parts Inventory",
		[]g.Node{
			H1(g.Text("Parts Pile")),
			P(g.Text("Welcome to Parts Pile - Your Electronic Parts Inventory System")),
		},
	).Render(w)
}

func Page(title string, content []g.Node) g.Node {
	return HTML(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(title),
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css")),
		),
		Body(
			Div(
				Class("container mx-auto px-4 py-8"),
				g.Group(content),
			),
		),
	)
}
