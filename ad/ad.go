package ad

import (
	"encoding/json"
	"os"
	"sync"
)

type Ad struct {
	ID          int      `json:"id"`
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
}

var (
	ads      = make(map[int]Ad)
	adsMutex sync.Mutex
	nextAdID = 1
)

// GetAllAds returns a copy of all ads
func GetAllAds() map[int]Ad {
	adsMutex.Lock()
	defer adsMutex.Unlock()

	// Create a copy to prevent external modification
	copy := make(map[int]Ad, len(ads))
	for k, v := range ads {
		copy[k] = v
	}
	return copy
}

// GetAd returns a specific ad by ID
func GetAd(id int) (Ad, bool) {
	adsMutex.Lock()
	defer adsMutex.Unlock()
	ad, ok := ads[id]
	return ad, ok
}

// AddAd adds a new ad and returns its ID
func AddAd(ad Ad) int {
	adsMutex.Lock()
	defer adsMutex.Unlock()

	ad.ID = nextAdID
	ads[ad.ID] = ad
	nextAdID++
	return ad.ID
}

// UpdateAd updates an existing ad
func UpdateAd(id int, ad Ad) bool {
	adsMutex.Lock()
	defer adsMutex.Unlock()

	if _, exists := ads[id]; !exists {
		return false
	}
	ad.ID = id
	ads[id] = ad
	return true
}

// DeleteAd deletes an ad by ID
func DeleteAd(id int) bool {
	adsMutex.Lock()
	defer adsMutex.Unlock()

	if _, exists := ads[id]; !exists {
		return false
	}
	delete(ads, id)
	return true
}

// GetNextAdID returns the next available ad ID
func GetNextAdID() int {
	adsMutex.Lock()
	defer adsMutex.Unlock()
	return nextAdID
}

// SetNextAdID sets the next available ad ID
func SetNextAdID(id int) {
	adsMutex.Lock()
	defer adsMutex.Unlock()
	nextAdID = id
}

// LoadAds loads ads from a JSON file and sets up the next available ID
func LoadAds(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, that's fine - we'll start with an empty map
			return nil
		}
		return err
	}

	adsMutex.Lock()
	defer adsMutex.Unlock()

	if err := json.Unmarshal(data, &ads); err != nil {
		return err
	}

	// Find max ID to set next ID
	maxID := 0
	for _, ad := range ads {
		if ad.ID > maxID {
			maxID = ad.ID
		}
	}
	nextAdID = maxID + 1

	return nil
}

// SaveAds saves ads to a JSON file
func SaveAds(filename string) error {
	adsMutex.Lock()
	defer adsMutex.Unlock()

	data, err := json.MarshalIndent(ads, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
