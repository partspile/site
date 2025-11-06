package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
)

type MakeParent struct {
	Make          string `json:"make"`
	ParentCompany string `json:"parent_company"`
}

type Parent struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

func main() {
	// Initialize logging
	config.InitLoggingConfig()
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	if len(os.Args) < 2 {
		logger.Error("Usage", "command", os.Args[0],
			"usage", "<make-year-model.json> [make-parent.json] [parent.json]",
			"description", "If make-parent.json and parent.json are provided, will analyze missing makes and parent companies")
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Error("Error reading file", "error", err, "filename", filename)
		os.Exit(1)
	}

	var makes map[string]interface{}
	if err := json.Unmarshal(data, &makes); err != nil {
		logger.Error("Error parsing JSON", "error", err, "filename", filename)
		os.Exit(1)
	}

	makeList := make([]string, 0, len(makes))
	for make := range makes {
		makeList = append(makeList, make)
	}
	sort.Strings(makeList)

	logger.Info("Total makes found", "filename", filename, "count", len(makeList))

	// If additional files are provided, analyze them
	if len(os.Args) >= 4 {
		analyzeMissingMakes(makeList, os.Args[2], os.Args[3])
	} else {
		// Just print the makes
		for _, make := range makeList {
			fmt.Println(make)
		}
	}
}

func analyzeMissingMakes(allMakes []string, makeParentFile, parentFile string) {
	// Read make-parent.json
	makeParentData, err := ioutil.ReadFile(makeParentFile)
	if err != nil {
		logger.Error("Error reading make-parent.json", "error", err, "filename", makeParentFile)
		return
	}

	var makeParents []MakeParent
	if err := json.Unmarshal(makeParentData, &makeParents); err != nil {
		logger.Error("Error parsing make-parent.json", "error", err, "filename", makeParentFile)
		return
	}

	// Read parent.json
	parentData, err := ioutil.ReadFile(parentFile)
	if err != nil {
		logger.Error("Error reading parent.json", "error", err, "filename", parentFile)
		return
	}

	var parents []Parent
	if err := json.Unmarshal(parentData, &parents); err != nil {
		logger.Error("Error parsing parent.json", "error", err, "filename", parentFile)
		return
	}

	// Create maps for easy lookup
	existingMakes := make(map[string]bool)
	existingParents := make(map[string]bool)

	for _, mp := range makeParents {
		existingMakes[mp.Make] = true
	}

	for _, p := range parents {
		existingParents[p.Name] = true
	}

	// Find missing makes
	var missingMakes []string
	for _, make := range allMakes {
		if !existingMakes[make] {
			missingMakes = append(missingMakes, make)
		}
	}

	logger.Info("Analysis results",
		"makesInFile", len(makeParents),
		"parentCompanies", len(parents),
		"missingMakes", len(missingMakes))

	if len(missingMakes) > 0 {
		fmt.Println("Missing makes:")
		for _, make := range missingMakes {
			fmt.Printf("  - %s\n", make)
		}
		fmt.Println()

		// Suggest parent companies for missing makes
		fmt.Println("Suggested parent companies to add:")
		suggestedParents := map[string]string{
			"ISO":    "Isotta Fraschini",
			"ISUZU":  "Isuzu Motors",
			"IVECO":  "CNH Industrial",
			"JAC":    "JAC Motors",
			"JAGUAR": "Jaguar Land Rover",
			"JEEP":   "Stellantis",
		}

		for _, make := range missingMakes {
			if parent, exists := suggestedParents[make]; exists {
				fmt.Printf("  - %s: %s\n", make, parent)
			} else {
				fmt.Printf("  - %s: [Need to research parent company]\n", make)
			}
		}
	}
}
