package part

type PartData map[string][]string

var (
	Data PartData
)

func GetAllCategories() []string {
	categories := make([]string, 0, len(Data))
	for category := range Data {
		categories = append(categories, category)
	}
	return categories
}

func GetAllSubCategories() []string {
	subCategories := make(map[string]struct{})
	for _, subs := range Data {
		for _, sub := range subs {
			subCategories[sub] = struct{}{}
		}
	}

	result := make([]string, 0, len(subCategories))
	for sub := range subCategories {
		result = append(result, sub)
	}
	return result
}
