package part

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	expectedCategories := []Category{
		{ID: 1, Name: "Engine"},
		{ID: 2, Name: "Brakes"},
		{ID: 3, Name: "Suspension"},
	}

	rows := sqlmock.NewRows([]string{"id", "name"})
	for _, category := range expectedCategories {
		rows.AddRow(category.ID, category.Name)
	}

	mock.ExpectQuery("SELECT id, name FROM PartCategory ORDER BY name").
		WillReturnRows(rows)

	categories, err := GetAllCategories()

	assert.NoError(t, err)
	assert.Len(t, categories, 3)
	assert.Equal(t, expectedCategories[0].Name, categories[0].Name)
	assert.Equal(t, expectedCategories[1].Name, categories[1].Name)
	assert.Equal(t, expectedCategories[2].Name, categories[2].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllSubCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	expectedSubCategories := []SubCategory{
		{ID: 1, CategoryID: 1, Name: "Engine Block"},
		{ID: 2, CategoryID: 1, Name: "Cylinder Head"},
		{ID: 3, CategoryID: 2, Name: "Brake Pads"},
	}

	rows := sqlmock.NewRows([]string{"id", "category_id", "name"})
	for _, subCategory := range expectedSubCategories {
		rows.AddRow(subCategory.ID, subCategory.CategoryID, subCategory.Name)
	}

	mock.ExpectQuery("SELECT id, category_id, name FROM PartSubCategory ORDER BY name").
		WillReturnRows(rows)

	subCategories, err := GetAllSubCategories()

	assert.NoError(t, err)
	assert.Len(t, subCategories, 3)
	assert.Equal(t, expectedSubCategories[0].Name, subCategories[0].Name)
	assert.Equal(t, expectedSubCategories[0].CategoryID, subCategories[0].CategoryID)
	assert.Equal(t, expectedSubCategories[1].Name, subCategories[1].Name)
	assert.Equal(t, expectedSubCategories[2].Name, subCategories[2].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMakes_WithQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	query := "engine"
	expectedMakes := []string{"BMW", "Mercedes", "Toyota"}

	rows := sqlmock.NewRows([]string{"name"})
	for _, make := range expectedMakes {
		rows.AddRow(make)
	}

	mock.ExpectQuery("SELECT DISTINCT m.name FROM Make m JOIN Car c ON m.id = c.make_id JOIN AdCar ac ON c.id = ac.car_id JOIN Ad a ON ac.ad_id = a.id WHERE a.description LIKE \\? ORDER BY m.name").
		WithArgs("%" + query + "%").
		WillReturnRows(rows)

	makes, err := GetMakes(query)

	assert.NoError(t, err)
	assert.Len(t, makes, 3)
	assert.Equal(t, expectedMakes[0], makes[0])
	assert.Equal(t, expectedMakes[1], makes[1])
	assert.Equal(t, expectedMakes[2], makes[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMakes_WithoutQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	expectedMakes := []string{"BMW", "Mercedes", "Toyota"}

	rows := sqlmock.NewRows([]string{"name"})
	for _, make := range expectedMakes {
		rows.AddRow(make)
	}

	mock.ExpectQuery("SELECT DISTINCT m.name FROM Make m JOIN Car c ON m.id = c.make_id JOIN AdCar ac ON c.id = ac.car_id ORDER BY m.name").
		WillReturnRows(rows)

	makes, err := GetMakes("")

	assert.NoError(t, err)
	assert.Len(t, makes, 3)
	assert.Equal(t, expectedMakes[0], makes[0])
	assert.Equal(t, expectedMakes[1], makes[1])
	assert.Equal(t, expectedMakes[2], makes[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}
