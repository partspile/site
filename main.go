package main

import (
	"log"

	"github.com/sfeldma/parts-pile/site/server"
)

func main() {
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
