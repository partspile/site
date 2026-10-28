package businesscard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

//go:embed data/ad-category.json
var embeddedCategoriesJSON []byte

// Category describes one listing category for card generation.
type Category struct {
	ID        int
	Name      string
	ImageFile string
}

type categoryJSON struct {
	Name      string `json:"name"`
	ImageFile string `json:"image_file"`
}

// LoadCategories reads ad-category.json and assigns stable IDs using
// alphabetical ordering by name. Pass an empty path to use the embedded list.
func LoadCategories(path string) ([]Category, error) {
	data := embeddedCategoriesJSON
	if path != "" {
		fileData, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		data = fileData
	}

	var raw []categoryJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	sort.Slice(raw, func(i, j int) bool {
		return raw[i].Name < raw[j].Name
	})

	cats := make([]Category, len(raw))
	for i, c := range raw {
		cats[i] = Category{
			ID:        i + 1,
			Name:      c.Name,
			ImageFile: c.ImageFile,
		}
	}
	return cats, nil
}

// FindCategory returns the category with the given name.
func FindCategory(cats []Category, name string) (Category, error) {
	for _, c := range cats {
		if c.Name == name {
			return c, nil
		}
	}
	return Category{}, fmt.Errorf("category not found: %s", name)
}

// CategoryURL returns the deep-link used on printed QR codes.
func CategoryURL(host string, categoryID int) string {
	return fmt.Sprintf("https://%s/c/%d", host, categoryID)
}
