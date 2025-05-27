package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type Engine []string        // list of engine sizes
type Year map[string]Engine // key: model name -> engine sizes
type Make map[string]*Year  // key: year -> models
type Makes map[string]*Make // key: make name -> years

// ScrapeRockAuto scrapes vehicle information from RockAuto.com and saves it to the specified file
func ScrapeRockAuto(outputFile string) error {
	vehicles := make(Makes)

	// Create a collector with custom settings
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	// Create separate collectors for years, models, and engines
	makeCollector := c.Clone()
	yearCollector := c.Clone()
	engineCollector := c.Clone()

	// Configure transport for all collectors
	c.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})
	makeCollector.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})
	yearCollector.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})
	engineCollector.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})

	// Configure rate limiting for all collectors
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*rockauto.com*",
		RandomDelay: 2 * time.Second,
		Parallelism: 1,
	})
	makeCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*rockauto.com*",
		RandomDelay: 2 * time.Second,
		Parallelism: 1,
	})
	yearCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*rockauto.com*",
		RandomDelay: 2 * time.Second,
		Parallelism: 1,
	})
	engineCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*rockauto.com*",
		RandomDelay: 2 * time.Second,
		Parallelism: 1,
	})

	// Handle errors for all collectors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", r.Request.URL, r, err)
	})
	makeCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", r.Request.URL, r, err)
	})
	yearCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", r.Request.URL, r, err)
	})
	engineCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", r.Request.URL, r, err)
	})

	// Debug: Print the HTML content when we get a response
	makeCollector.OnResponse(func(r *colly.Response) {
		log.Printf("Received response from URL: %s", r.Request.URL.String())
		log.Printf("Looking for year data in HTML...")
	})

	yearCollector.OnResponse(func(r *colly.Response) {
		log.Printf("Received response from URL: %s", r.Request.URL.String())
		log.Printf("Looking for model data in HTML...")
	})

	engineCollector.OnResponse(func(r *colly.Response) {
		log.Printf("Received response from URL: %s", r.Request.URL.String())
		log.Printf("Looking for engine size data in HTML...")
	})

	// Helper function to write current state to JSON file
	writeJSON := func() error {
		vehiclesData, err := json.MarshalIndent(vehicles, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal vehicles JSON: %v", err)
		}

		err = os.WriteFile(outputFile, vehiclesData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write vehicles JSON file: %v", err)
		}
		return nil
	}

	// Extract makes from the main navigation menu
	c.OnHTML("div.ranavnode[id^='nav[']", func(e *colly.HTMLElement) {
		// Get the hidden input with make data
		makeData := e.ChildAttr("input[id^='jsn[']", "value")
		if !strings.Contains(makeData, "nodetype\":\"make\"") {
			return
		}

		// Get the make name from the link
		makeName := e.ChildText("a.navlabellink")
		if makeName == "" || makeName == "Select a Make" || makeName == "--" {
			return
		}

		makeName = strings.ToUpper(makeName)
		log.Printf("Found make: %s", makeName)

		// Initialize make if it doesn't exist
		if _, exists := vehicles[makeName]; !exists {
			vehicles[makeName] = new(Make)
			*vehicles[makeName] = make(Make)
		}

		// Get the make's URL and visit it to get years
		makeLink := e.ChildAttr("a.navlabellink", "href")
		if makeLink != "" {
			makeURL := fmt.Sprintf("https://www.rockauto.com%s", makeLink)
			log.Printf("Visiting make URL for %s: %s", makeName, makeURL)
			err := makeCollector.Visit(makeURL)
			if err != nil {
				log.Printf("Error visiting make URL for %s: %v", makeName, err)
			}
		}

		// Write current state to JSON file after processing each make
		if err := writeJSON(); err != nil {
			log.Printf("Error writing JSON after processing make %s: %v", makeName, err)
		} else {
			log.Printf("Successfully wrote current state to JSON after processing make: %s", makeName)
		}
	})

	// Extract years from the navigation nodes
	makeCollector.OnHTML("div.ranavnode[id^='nav[']", func(e *colly.HTMLElement) {
		// Get the make name from the URL
		urlParts := strings.Split(e.Request.URL.Path, "/")
		if len(urlParts) < 4 {
			return
		}

		makeName := urlParts[3]
		makeName = strings.ReplaceAll(makeName, "+", " ")
		makeName = strings.ToUpper(makeName)

		// Get the node data
		nodeData := e.ChildAttr("input[id^='jsn[']", "value")
		if nodeData == "" {
			return
		}

		// Check if this is a year node
		if strings.Contains(nodeData, "nodetype\":\"year\"") {
			year := e.ChildText("a.navlabellink")
			if year == "" || year == "Select a Year" || year == "--" {
				return
			}

			log.Printf("Found year node: %s for make %s", year, makeName)

			if makeData, ok := vehicles[makeName]; ok {
				// Initialize year if it doesn't exist
				if _, exists := (*makeData)[year]; !exists {
					(*makeData)[year] = new(Year)
					*(*makeData)[year] = make(Year)
					log.Printf("Added year %s to make %s", year, makeName)

					// Visit the year's page to get models
					yearLink := e.ChildAttr("a.navlabellink", "href")
					if yearLink != "" {
						yearURL := fmt.Sprintf("https://www.rockauto.com%s", yearLink)
						log.Printf("Visiting year URL for %s %s: %s", makeName, year, yearURL)
						err := yearCollector.Visit(yearURL)
						if err != nil {
							log.Printf("Error visiting year URL for %s %s: %v", makeName, year, err)
						}
					}
				}
			}
		}
	})

	// Extract models from the navigation nodes
	yearCollector.OnHTML("div.ranavnode", func(e *colly.HTMLElement) {
		// Get the make and year from the URL using the comma format
		urlPath := e.Request.URL.Path
		log.Printf("Processing URL path: %s", urlPath)

		// Remove the /en/catalog/ prefix
		urlPath = strings.TrimPrefix(urlPath, "/en/catalog/")

		// Split by comma to separate make and year
		parts := strings.Split(urlPath, ",")
		if len(parts) < 2 {
			log.Printf("URL path doesn't contain make,year format: %s", urlPath)
			return
		}

		makeName := parts[0]
		year := parts[1]

		makeName = strings.ReplaceAll(makeName, "+", " ")
		makeName = strings.ToUpper(makeName)

		// Get the node data
		nodeData := e.ChildAttr("input[id^='jsn[']", "value")
		if nodeData == "" {
			return
		}

		// Debug node data
		log.Printf("Examining node data for make %s year %s: %s", makeName, year, nodeData)

		// Check if this is a model node - try both conditions
		if strings.Contains(nodeData, "nodetype\":\"model\"") || strings.Contains(nodeData, "\"model\"") {
			model := e.ChildText("a.navlabellink")
			if model == "" || model == "Select a Model" || model == "--" {
				return
			}

			// Standardize model name to uppercase
			model = strings.ToUpper(model)

			log.Printf("Found model node: %s for make %s year %s", model, makeName, year)

			if makeData, ok := vehicles[makeName]; ok {
				if yearData, ok := (*makeData)[year]; ok {
					// Initialize engine if it doesn't exist
					if _, exists := (*yearData)[model]; !exists {
						(*yearData)[model] = make(Engine, 0)
						log.Printf("Added model %s to make %s year %s", model, makeName, year)

						// Visit the model's page to get engine sizes
						modelLink := e.ChildAttr("a.navlabellink", "href")
						if modelLink != "" {
							modelURL := fmt.Sprintf("https://www.rockauto.com%s", modelLink)
							log.Printf("Visiting model URL for %s %s %s: %s", makeName, year, model, modelURL)
							err := engineCollector.Visit(modelURL)
							if err != nil {
								log.Printf("Error visiting model URL for %s %s %s: %v", makeName, year, model, err)
							}
						}
					}
				}
			}
		}
	})

	// Extract engine sizes from the navigation nodes
	engineCollector.OnHTML("div.ranavnode", func(e *colly.HTMLElement) {
		// Get the make, year, and model from the URL
		urlPath := e.Request.URL.Path
		log.Printf("Processing URL path for engine: %s", urlPath)

		// Remove the /en/catalog/ prefix
		urlPath = strings.TrimPrefix(urlPath, "/en/catalog/")

		// Split by comma to separate make, year, and model
		parts := strings.Split(urlPath, ",")
		if len(parts) < 3 {
			log.Printf("URL path doesn't contain make,year,model format: %s", urlPath)
			return
		}

		makeName := parts[0]
		year := parts[1]
		model := parts[2]

		makeName = strings.ReplaceAll(makeName, "+", " ")
		makeName = strings.ToUpper(makeName)
		model = strings.ReplaceAll(model, "+", " ")
		model = strings.ToUpper(model)

		// Get the node data
		nodeData := e.ChildAttr("input[id^='jsn[']", "value")
		if nodeData == "" {
			return
		}

		// Debug node data
		log.Printf("Examining engine node data for make %s year %s model %s: %s", makeName, year, model, nodeData)

		// Check if this is an engine node
		if strings.Contains(nodeData, "nodetype\":\"engine\"") || strings.Contains(nodeData, "\"engine\"") {
			engineSize := e.ChildText("a.navlabellink")
			if engineSize == "" || engineSize == "Select an Engine" || engineSize == "--" {
				return
			}

			log.Printf("Found engine node: %s for make %s year %s model %s", engineSize, makeName, year, model)

			if makeData, ok := vehicles[makeName]; ok {
				if yearData, ok := (*makeData)[year]; ok {
					if engineSizes, ok := (*yearData)[model]; ok {
						// Check for duplicates
						duplicate := false
						for _, existingSize := range engineSizes {
							if existingSize == engineSize {
								duplicate = true
								break
							}
						}
						if !duplicate {
							(*yearData)[model] = append(engineSizes, engineSize)
							log.Printf("Added engine size %s to make %s year %s model %s (total sizes: %d)",
								engineSize, makeName, year, model, len((*yearData)[model]))
						}
					}
				}
			}
		}
	})

	// Visit the main page to start scraping makes
	log.Printf("Starting to scrape makes, years, models, and engine sizes from RockAuto.com...")
	err := c.Visit("https://www.rockauto.com/en/catalog/")
	if err != nil {
		return fmt.Errorf("failed to visit rockauto.com: %v", err)
	}

	// Wait for all collectors to complete
	c.Wait()
	makeCollector.Wait()
	yearCollector.Wait()
	engineCollector.Wait()

	log.Printf("Scraping complete. Writing final data to file...")

	// Write final state
	return writeJSON()
}
