package main

import (
	"fmt"
	"log"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
)

func buildAdEmbeddingPrompt(adObj ad.Ad) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyStr = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	return fmt.Sprintf(`Encode the following ad for semantic search. Focus on what the part is, what vehicles it fits, and any relevant details for a buyer. Return only the embedding vector.\n\nTitle: %s\nDescription: %s\nMake: %s\nParent Company: %s\nParent Company Country: %s\nYears: %s\nModels: %s\nEngines: %s\nCategory: %s\nSubCategory: %s\nLocation: %s, %s, %s`,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.Category,
		adObj.SubCategory,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
	)
}

func buildAdEmbeddingMetadata(adObj ad.Ad) map[string]interface{} {
	// Get parent company information for the make
	var parentCompanyName, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyName = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	return map[string]interface{}{
		"ad_id":                  adObj.ID,
		"created_at":             adObj.CreatedAt.Format(time.RFC3339),
		"click_count":            adObj.ClickCount,
		"make":                   adObj.Make,
		"parent_company":         parentCompanyName,
		"parent_company_country": parentCompanyCountry,
		"years":                  interfaceSlice(adObj.Years),
		"models":                 interfaceSlice(adObj.Models),
		"engines":                interfaceSlice(adObj.Engines),
		"category":               adObj.Category,
		"subcategory":            adObj.SubCategory,
		"city":                   adObj.City,
		"admin_area":             adObj.AdminArea,
		"country":                adObj.Country,
	}
}

func interfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

func main() {
	if err := ad.InitDB("project.db"); err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	// Initialize vehicle package with the same DB connection
	vehicle.InitDB(ad.DB)
	// Initialize Gemini and Pinecone clients
	if err := vector.InitGeminiClient(""); err != nil {
		log.Fatalf("Failed to init Gemini: %v", err)
	}
	if err := vector.InitPineconeClient("", ""); err != nil {
		log.Fatalf("Failed to init Pinecone: %v", err)
	}
	ads, err := ad.GetAllAds()
	if err != nil {
		log.Fatalf("Failed to get ads: %v", err)
	}
	fmt.Printf("Found %d ads to backfill\n", len(ads))
	failures := 0
	for i, adObj := range ads {
		prompt := buildAdEmbeddingPrompt(adObj)
		log.Printf("[backfill] Generating embedding for ad %d with prompt: %s", adObj.ID, prompt)
		embedding, err := vector.EmbedText(prompt)
		if err != nil {
			log.Printf("[embedding] failed for ad %d: %v", adObj.ID, err)
			failures++
			continue
		}
		meta := buildAdEmbeddingMetadata(adObj)
		err = vector.UpsertAdEmbedding(adObj.ID, embedding, meta)
		if err != nil {
			log.Printf("[pinecone] upsert failed for ad %d: %v", adObj.ID, err)
			failures++
			continue
		}
		if (i+1)%10 == 0 || i == len(ads)-1 {
			fmt.Printf("%d/%d ads processed\n", i+1, len(ads))
		}
		// Sleep to avoid rate limits if needed
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("Backfill complete. %d ads processed, %d failures.\n", len(ads), failures)
}
