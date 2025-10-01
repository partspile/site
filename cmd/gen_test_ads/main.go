package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/parts-pile/site/grok"
)

// Data structures for loading JSON files
type MakeYearModelData map[string]map[string]map[string][]string

type PartData map[string][]string

type Location struct {
	City      string  `json:"city"`
	AdminArea string  `json:"admin_area"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Ad struct {
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	CreatedAt   string   `json:"created_at"`
	UserID      int      `json:"user_id"`
	Category    string   `json:"category"`
	Subcategory string   `json:"subcategory"`
	Location    Location `json:"location"`
}

func main() {
	var (
		count     = flag.Int("count", 30, "Number of ads to generate")
		output    = flag.String("output", "", "Output file (default: stdout)")
		seed      = flag.Int64("seed", time.Now().UnixNano(), "Random seed")
		userID    = flag.Int("user-id", 1, "User ID for generated ads")
		startDate = flag.String("start-date", "2024-01-01", "Start date for created_at (YYYY-MM-DD)")
		workers   = flag.Int("workers", 8, "Number of parallel workers (max 8 to respect rate limits)")
		debug     = flag.Bool("debug", false, "Enable debug output (shows Grok API requests/responses)")
	)
	flag.Parse()

	// Limit workers to respect rate limits
	if *workers > 8 {
		*workers = 8
	}
	if *workers < 1 {
		*workers = 1
	}

	// Set random seed
	rand.Seed(*seed)

	// Load data files
	makeYearModelData, err := loadMakeYearModelData()
	if err != nil {
		log.Fatalf("Failed to load make-year-model.json: %v", err)
	}

	partData, err := loadPartData()
	if err != nil {
		log.Fatalf("Failed to load part.json: %v", err)
	}

	// Parse start date
	startTime, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}

	// Generate ads in parallel
	ads := generateAdsParallel(makeYearModelData, partData, *userID, startTime, *count, *workers, *debug)

	// Output results
	var outputData []byte
	if *output == "" {
		// Stream output as JSON array
		outputData, err = json.MarshalIndent(ads, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal ads: %v", err)
		}
		fmt.Println(string(outputData))
	} else {
		// Write to file
		outputData, err = json.MarshalIndent(ads, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal ads: %v", err)
		}
		err = os.WriteFile(*output, outputData, 0644)
		if err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Generated %d ads and saved to %s\n", len(ads), *output)
	}
}

func loadMakeYearModelData() (MakeYearModelData, error) {
	// Try multiple possible paths
	paths := []string{
		"../../cmd/rebuild_db/make-year-model.json",
		"cmd/rebuild_db/make-year-model.json",
		"./cmd/rebuild_db/make-year-model.json",
	}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("could not find make-year-model.json in any of the expected locations: %v", err)
	}

	var result MakeYearModelData
	err = json.Unmarshal(data, &result)
	return result, err
}

func loadPartData() (PartData, error) {
	// Try multiple possible paths
	paths := []string{
		"../../cmd/rebuild_db/part.json",
		"cmd/rebuild_db/part.json",
		"./cmd/rebuild_db/part.json",
	}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("could not find part.json in any of the expected locations: %v", err)
	}

	var result PartData
	err = json.Unmarshal(data, &result)
	return result, err
}

func selectWeightedMake(makeData MakeYearModelData) string {
	// Define popular American makes with much higher weights
	popularAmericanMakes := map[string]int{
		"FORD":       200, // Most popular
		"CHEVROLET":  180,
		"DODGE":      160,
		"CHRYSLER":   140,
		"BUICK":      120,
		"CADILLAC":   120,
		"GMC":        120,
		"JEEP":       120,
		"LINCOLN":    100,
		"MERCURY":    100,
		"PONTIAC":    100,
		"OLDSMOBILE": 80,
		"PLYMOUTH":   80,
		"RAM":        80,
		"HUMMER":     60,
		"SATURN":     60,
		"TESLA":      60,
	}

	// Define popular foreign makes with lower weights
	popularForeignMakes := map[string]int{
		"BMW":         8,
		"AUDI":        8,
		"MERCEDES":    8,
		"VOLKSWAGEN":  8,
		"TOYOTA":      8,
		"HONDA":       8,
		"NISSAN":      8,
		"MAZDA":       5,
		"SUBARU":      5,
		"INFINITI":    5,
		"ACURA":       5,
		"LEXUS":       5,
		"PORSCHE":     3,
		"FERRARI":     2,
		"LAMBORGHINI": 2,
		"MASERATI":    2,
		"JAGUAR":      2,
		"BENTLEY":     2,
		"ROLLS":       2,
		"ASTON":       2,
		"MCLAREN":     2,
		"LOTUS":       2,
		"MINI":        2,
		"LAND":        2,
		"RANGE":       2,
		"ALFA":        2,
		"FIAT":        2,
		"ABARTH":      2,
	}

	// Create weighted list
	var weightedMakes []string
	var totalWeight int

	// Add all makes with their weights
	for make := range makeData {
		if weight, exists := popularAmericanMakes[make]; exists {
			// Add popular American makes multiple times based on weight
			for i := 0; i < weight; i++ {
				weightedMakes = append(weightedMakes, make)
				totalWeight += weight
			}
		} else if weight, exists := popularForeignMakes[make]; exists {
			// Add popular foreign makes with medium weight
			for i := 0; i < weight; i++ {
				weightedMakes = append(weightedMakes, make)
				totalWeight += weight
			}
		} else {
			// Add lesser-known makes with low weight (1-3)
			weight := rand.Intn(3) + 1
			for i := 0; i < weight; i++ {
				weightedMakes = append(weightedMakes, make)
				totalWeight += weight
			}
		}
	}

	// Select random make from weighted list
	if len(weightedMakes) == 0 {
		// Fallback to random selection if no weighted makes
		makes := make([]string, 0, len(makeData))
		for make := range makeData {
			makes = append(makes, make)
		}
		return makes[rand.Intn(len(makes))]
	}

	return weightedMakes[rand.Intn(len(weightedMakes))]
}

func generateAdsParallel(makeData MakeYearModelData, partData PartData, userID int, startTime time.Time, count, workers int, debug bool) []Ad {
	// Create jobs channel
	jobs := make(chan int, count)
	results := make(chan Ad, count)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Rate limiting: 480 requests per minute = 8 per second
			// With 8 workers, each worker should wait ~1 second between requests
			ticker := time.NewTicker(125 * time.Millisecond) // 8 requests per second
			defer ticker.Stop()

			for jobID := range jobs {
				<-ticker.C // Wait for rate limit
				ad, err := generateAd(makeData, partData, userID, startTime, jobID, debug)
				if err != nil {
					log.Printf("Worker %d failed to generate ad %d: %v", workerID, jobID+1, err)
					continue
				}
				results <- ad
				fmt.Printf("Ad %d/%d: %s %s %s (%s) -> %s\n", jobID+1, count, ad.Make, strings.Join(ad.Years, ","), strings.Join(ad.Models, ","), strings.Join(ad.Engines, ","), ad.Title)
			}
		}(i)
	}

	// Send jobs
	for i := 0; i < count; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results
	ads := make([]Ad, 0, count)
	for ad := range results {
		ads = append(ads, ad)
	}

	return ads
}

func generateAd(makeData MakeYearModelData, partData PartData, userID int, startTime time.Time, index int, debug bool) (Ad, error) {
	// Try up to 10 times to find a fully specified combination
	for attempt := 0; attempt < 10; attempt++ {
		ad, err := generateAdAttempt(makeData, partData, userID, startTime, index, debug)
		if err == nil {
			return ad, nil
		}
		if debug {
			fmt.Printf("Attempt %d failed: %v, retrying...\n", attempt+1, err)
		}
	}
	return Ad{}, fmt.Errorf("failed to generate valid ad after 10 attempts")
}

func generateAdAttempt(makeData MakeYearModelData, partData PartData, userID int, startTime time.Time, index int, debug bool) (Ad, error) {
	// Weighted selection of makes (favor popular American makes)
	selectedMake := selectWeightedMake(makeData)

	// Validate make is not empty
	if selectedMake == "" {
		return Ad{}, fmt.Errorf("empty make selected")
	}

	// Randomly select 1-3 years
	allYears := make([]string, 0, len(makeData[selectedMake]))
	for year := range makeData[selectedMake] {
		if year != "" { // Skip empty years
			allYears = append(allYears, year)
		}
	}

	if len(allYears) == 0 {
		return Ad{}, fmt.Errorf("no valid years found for make %s", selectedMake)
	}

	numYears := rand.Intn(3) + 1 // 1-3 years
	if numYears > len(allYears) {
		numYears = len(allYears)
	}

	selectedYears := make([]string, 0, numYears)
	yearIndices := rand.Perm(len(allYears))[:numYears]
	for _, idx := range yearIndices {
		selectedYears = append(selectedYears, allYears[idx])
	}

	// Find all models available across selected years
	availableModels := make(map[string]bool)
	for _, year := range selectedYears {
		for model := range makeData[selectedMake][year] {
			if model != "" { // Skip empty models
				availableModels[model] = true
			}
		}
	}

	if len(availableModels) == 0 {
		return Ad{}, fmt.Errorf("no valid models found for make %s and years %v", selectedMake, selectedYears)
	}

	// Randomly select 1-3 models from available models
	modelList := make([]string, 0, len(availableModels))
	for model := range availableModels {
		modelList = append(modelList, model)
	}

	numModels := rand.Intn(3) + 1 // 1-3 models
	if numModels > len(modelList) {
		numModels = len(modelList)
	}

	selectedModels := make([]string, 0, numModels)
	modelIndices := rand.Perm(len(modelList))[:numModels]
	for _, idx := range modelIndices {
		selectedModels = append(selectedModels, modelList[idx])
	}

	// Find all engines available across selected years and models
	availableEngines := make(map[string]bool)
	for _, year := range selectedYears {
		for _, model := range selectedModels {
			if engines, exists := makeData[selectedMake][year][model]; exists {
				for _, engine := range engines {
					if engine != "" { // Skip empty engines
						availableEngines[engine] = true
					}
				}
			}
		}
	}

	if len(availableEngines) == 0 {
		return Ad{}, fmt.Errorf("no valid engines found for make %s, years %v, models %v", selectedMake, selectedYears, selectedModels)
	}

	// Convert to slice
	engines := make([]string, 0, len(availableEngines))
	for engine := range availableEngines {
		engines = append(engines, engine)
	}

	// Randomly select category and subcategory
	categories := make([]string, 0, len(partData))
	for category := range partData {
		categories = append(categories, category)
	}
	selectedCategory := categories[rand.Intn(len(categories))]

	var selectedSubcategory string
	if subcategories := partData[selectedCategory]; len(subcategories) > 0 {
		selectedSubcategory = subcategories[rand.Intn(len(subcategories))]
	}

	// Generate ad content using Grok
	title, description, price, location, err := generateAdContent(selectedMake, selectedYears, selectedModels, engines, selectedCategory, selectedSubcategory, debug)
	if err != nil {
		return Ad{}, err
	}

	// Generate created_at timestamp
	daysOffset := rand.Intn(365) // Random day within a year
	createdAt := startTime.AddDate(0, 0, daysOffset).Format(time.RFC3339)

	return Ad{
		Make:        selectedMake,
		Years:       selectedYears,
		Models:      selectedModels,
		Engines:     engines,
		Title:       title,
		Description: description,
		Price:       price,
		CreatedAt:   createdAt,
		UserID:      userID,
		Category:    selectedCategory,
		Subcategory: selectedSubcategory,
		Location:    location,
	}, nil
}

func generateAdContent(make string, years, models, engines []string, category, subcategory string, debug bool) (string, string, float64, Location, error) {
	// Create prompt for Grok
	systemPrompt := `You are an expert at creating realistic automotive parts advertisements. Generate content that sounds authentic and professional.`

	// Show what we're generating
	fmt.Printf("Generating: %s %s %s (%s)\n", make, strings.Join(years, ","), strings.Join(models, ","), strings.Join(engines, ","))

	userPrompt := fmt.Sprintf(`Create a realistic automotive parts advertisement with the following details:

Vehicle: %s %s %s
Engines: %s
Category: %s
Subcategory: %s

Please provide the response in this exact JSON format:
{
  "title": "Part name and vehicle info",
  "description": "Detailed description of the part, condition, compatibility, and what's included",
  "price": 123.45,
  "location": {
    "city": "City name",
    "admin_area": "State/Province/Region",
    "country": "Country code (US, DE, IT, GB, etc.)",
    "latitude": 40.7128,
    "longitude": -74.0060
  }
}

Create a concise title that mentions the part name and make, plus year range if multiple years (e.g., "BMW Oil Filter Housing" or "BMW 1955-1956 Radiator"). Only include specific model names if they're well-known or if there's only one model. Do NOT include engine details in the title. The description should be 2-3 sentences focusing on condition, compatibility, and what's included - don't repeat all the vehicle details. Price should be realistic for the part type and vehicle age. Location should be a real city appropriate for the vehicle make (German cities for BMW/Audi, Italian for Abarth, British for Austin/Bentley, etc.).`,
		make, strings.Join(years, ", "), strings.Join(models, ", "), strings.Join(engines, ", "), category, subcategory)

	response, err := grok.CallGrokWithDebug(systemPrompt, userPrompt, debug)
	if err != nil {
		return "", "", 0, Location{}, err
	}

	// Parse the JSON response
	var result struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Price       float64  `json:"price"`
		Location    Location `json:"location"`
	}

	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		return "", "", 0, Location{}, fmt.Errorf("failed to parse Grok response: %w", err)
	}

	return result.Title, result.Description, result.Price, result.Location, nil
}
