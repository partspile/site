package main

import (
	"log"

	"github.com/parts-pile/site/server"
)

func main() {
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
