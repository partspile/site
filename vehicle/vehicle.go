package vehicle

import (
	"encoding/json"
	"os"
	"slices"
	"sort"
	"sync"

	"github.com/parts-pile/site/ad"
)

type VehicleData map[string]map[string]map[string][]string

var (
	Data     VehicleData
	Ads      = make(map[int]ad.Ad)
	AdsMutex sync.Mutex
	NextAdID = 1
)

func GetMakes() []string {
	makesList := make([]string, 0, len(Data))
	for makeName := range Data {
		makesList = append(makesList, makeName)
	}
	sort.Strings(makesList)
	return makesList
}

func GetYears(makeName string) []string {
	years := make([]string, 0)
	if makeData, ok := Data[makeName]; ok {
		for year := range makeData {
			years = append(years, year)
		}
	}
	sort.Strings(years)
	return years
}

func GetAllModels() []string {
	models := make([]string, 0)
	for _, makeData := range Data {
		for _, yearData := range makeData {
			for model := range yearData {
				models = append(models, model)
			}
		}
	}
	sort.Strings(models)
	return slices.Compact(models)
}

func GetModelsWithAvailability(makeName string, years []string) map[string]bool {
	allModels := make(map[string]bool)
	availableInAllYears := make(map[string]bool)

	if makeData, ok := Data[makeName]; ok {
		// First pass: collect all models and mark them as potentially available
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for model := range yearData {
					allModels[model] = true
					if _, exists := availableInAllYears[model]; !exists {
						availableInAllYears[model] = true
					}
				}
			}
		}

		// Second pass: check if each model exists in all selected years
		for model := range allModels {
			for _, year := range years {
				if yearData, ok := makeData[year]; ok {
					if _, hasModel := yearData[model]; !hasModel {
						availableInAllYears[model] = false
						break
					}
				}
			}
		}
	}
	return availableInAllYears
}

func GetEnginesWithAvailability(makeName string, years []string, models []string) map[string]bool {
	allEngines := make(map[string]bool)
	availableInAllCombos := make(map[string]bool)

	if makeData, ok := Data[makeName]; ok {
		// First pass: collect all engines
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for _, model := range models {
					if engines, ok := yearData[model]; ok {
						for _, engine := range engines {
							allEngines[engine] = true
							if _, exists := availableInAllCombos[engine]; !exists {
								availableInAllCombos[engine] = true
							}
						}
					}
				}
			}
		}

		// Second pass: check if each engine exists for all selected year-model combinations
		for engine := range allEngines {
			for _, year := range years {
				if yearData, ok := makeData[year]; ok {
					for _, model := range models {
						if engines, ok := yearData[model]; ok {
							engineFound := false
							for _, e := range engines {
								if e == engine {
									engineFound = true
									break
								}
							}
							if !engineFound {
								availableInAllCombos[engine] = false
								break
							}
						} else {
							availableInAllCombos[engine] = false
							break
						}
					}
					if !availableInAllCombos[engine] {
						break
					}
				}
			}
		}
	}
	return availableInAllCombos
}

func GetAllEngineSizes() []string {
	engines := make([]string, 0)
	for _, makeData := range Data {
		for _, yearData := range makeData {
			for _, enginesList := range yearData {
				engines = append(engines, enginesList...)
			}
		}
	}
	sort.Strings(engines)
	return slices.Compact(engines)
}

func LoadData() error {
	data, err := os.ReadFile("make-year-model.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &Data)
}
