package handlers

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/rock"
	"github.com/parts-pile/site/ui"
)

// HandleAdRocks displays the rock section for an ad
func HandleAdRocks(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid ad ID")
	}

	currentUser, err := GetCurrentUser(c)
	if err != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get rock count for the ad
	rockCount, err := rock.GetAdRockCount(adID)
	if err != nil {
		return c.Status(500).SendString("Failed to get rock count")
	}

	// Check if current user can throw a rock
	canThrow := false
	if currentUser != nil {
		canThrow, err = rock.CanThrowRock(currentUser.ID)
		if err != nil {
			canThrow = false
		}
	}

	// Get user's rock count
	userRockCount := 0
	if currentUser != nil {
		userRocks, err := rock.GetUserRocks(currentUser.ID)
		if err == nil {
			userRockCount = userRocks.RockCount
		}
	}

	// Render rock section
	rockSection := ui.RockSection(adID, rockCount, canThrow, userRockCount)
	return render(c, rockSection)
}

// HandleThrowRock handles throwing a rock at an ad
func HandleThrowRock(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid ad ID")
	}

	currentUser, err := GetCurrentUser(c)
	if err != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get the initial message from the form
	message := c.FormValue("message")
	if message == "" {
		message = "I'm throwing a rock at this ad due to concerns about quality, accuracy, or policy violations."
	}

	// Throw the rock
	err = rock.ThrowRock(currentUser.ID, adID, message)
	if err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to throw rock: %v", err))
	}

	// Get updated rock count
	rockCount, err := rock.GetAdRockCount(adID)
	if err != nil {
		rockCount = 0
	}

	// Check if user can still throw rocks
	canThrow, err := rock.CanThrowRock(currentUser.ID)
	if err != nil {
		canThrow = false
	}

	// Get user's updated rock count
	userRocks, err := rock.GetUserRocks(currentUser.ID)
	userRockCount := 0
	if err == nil {
		userRockCount = userRocks.RockCount
	}

	// Render updated rock section
	rockSection := ui.RockSection(adID, rockCount, canThrow, userRockCount)
	return render(c, rockSection)
}

// HandleViewRockConversations displays the rock conversations for an ad
func HandleViewRockConversations(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid ad ID")
	}

	// Get all rocks for the ad
	rocks, err := rock.GetAdRocks(adID)
	if err != nil {
		return c.Status(500).SendString("Failed to get rocks")
	}

	// Render rock conversations
	conversations := ui.RockConversations(adID, rocks)
	return render(c, conversations)
}

// HandleResolveRock resolves a rock dispute
func HandleResolveRock(c *fiber.Ctx) error {
	rockID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid rock ID")
	}

	currentUser, err := GetCurrentUser(c)
	if err != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	// Resolve the rock
	err = rock.ResolveRock(rockID, currentUser.ID)
	if err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to resolve rock: %v", err))
	}

	return c.SendString("Rock resolved successfully")
}

// applyRockPenalties reorders search results to penalize ads with rocks
func applyRockPenalties(ads []ad.Ad) []ad.Ad {
	if len(ads) == 0 {
		return ads
	}

	// Create a copy to avoid modifying the original slice
	result := make([]ad.Ad, len(ads))
	copy(result, ads)

	// Sort by rock count (ascending) and then by creation date (descending)
	// This pushes ads with more rocks down in the results
	sort.Slice(result, func(i, j int) bool {
		// Get rock counts
		rockCountI, _ := rock.GetAdRockCount(result[i].ID)
		rockCountJ, _ := rock.GetAdRockCount(result[j].ID)

		// If rock counts are different, sort by rock count (ascending)
		if rockCountI != rockCountJ {
			return rockCountI < rockCountJ
		}

		// If rock counts are the same, sort by creation date (newer first)
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}
