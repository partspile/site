package main

import (
	"flag"
	"fmt"
	"log"

	"parts-pile.com/pkg/scraper"
)

func main() {
	outputFile := flag.String("output", "make-year-model.json", "Output JSON file path")
	flag.Parse()

	fmt.Printf("Starting to scrape RockAuto.com. Data will be saved to %s\n", *outputFile)
	if err := scraper.ScrapeRockAuto(*outputFile); err != nil {
		log.Fatalf("Failed to scrape RockAuto: %v", err)
	}
	fmt.Printf("Scraping completed. Data saved to %s\n", *outputFile)
}
