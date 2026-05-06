package repository

import (
	"comparify/internal/model"
	"errors"
	"fmt"
	"strings"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	ListByFilters(filters map[string]string) ([]model.Product, error)
	GetByID(id string) (model.Product, error)
	GetSpecificationsByType(productID, productType string) (map[string]string, error)
	GetSpecificationsBatch(ids []string, productType string) (map[string]map[string]string, error)
}

type InMemoryRepository struct {
	products map[string]model.Product
	order    []string
}

func NewInMemoryRepository(products []model.Product) *InMemoryRepository {
	repo := &InMemoryRepository{
		products: make(map[string]model.Product, len(products)),
		order:    make([]string, 0, len(products)),
	}

	for _, product := range products {
		repo.products[product.ID] = product
		repo.order = append(repo.order, product.ID)
	}

	return repo
}

func (r *InMemoryRepository) ListByFilters(filters map[string]string) ([]model.Product, error) {
	var result []model.Product
	for _, productID := range r.order {
		candidate := r.products[productID]
		allFiltersMatch := true
		for filterKey, filterValue := range filters {
			acceptedValues := strings.Split(filterValue, ",")
			productFieldValue := ""
			switch filterKey {
			case "color":
				productFieldValue = candidate.Color
			case "brand":
				productFieldValue = candidate.Specifications["brand"]
			}
			valueFound := false
			for _, acceptedValue := range acceptedValues {
				if strings.EqualFold(strings.TrimSpace(acceptedValue), productFieldValue) {
					valueFound = true
					break
				}
			}
			if !valueFound {
				allFiltersMatch = false
				break
			}
		}
		if allFiltersMatch {
			result = append(result, candidate)
		}
	}
	if len(result) == 0 {
		return nil, ErrProductNotFound
	}
	return result, nil
}

func (r *InMemoryRepository) ListByIDs(ids []string) ([]model.Product, error) {
	products := make([]model.Product, 0, len(ids))
	for _, id := range ids {
		product, err := r.GetByID(id)
		if err != nil {
			return nil, fmt.Errorf("lookup %q: %w", id, err)
		}
		products = append(products, product)
	}

	return products, nil
}

func (r *InMemoryRepository) GetByID(id string) (model.Product, error) {
	product, ok := r.products[id]
	if !ok {
		return model.Product{}, ErrProductNotFound
	}

	return product, nil
}

func (r *InMemoryRepository) GetSpecificationsByType(productID, productType string) (map[string]string, error) {
	product, ok := r.products[productID]
	if !ok {
		return nil, ErrProductNotFound
	}
	return product.Specifications, nil
}

func (r *InMemoryRepository) GetSpecificationsBatch(ids []string, productType string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string, len(ids))
	for _, id := range ids {
		p, ok := r.products[id]
		if ok {
			result[id] = p.Specifications
		}
	}
	return result, nil
}

func SeedProducts() []model.Product {
	return []model.Product{
		{
			ID:          "phone-1",
			Name:        "Atlas One",
			ImageURL:    "https://example.com/images/atlas-one.png",
			Description: "Smartphone focused on battery life and travel photography.",
			Price:       3299.90,
			Rating:      4.7,
			Size:        "161.2 x 74.5 x 8.1 mm",
			Weight:      "189 g",
			Color:       "Graphite",
			Specifications: map[string]string{
				"batteryCapacity":      "5000 mAh",
				"cameraSpecifications": "50 MP wide + 12 MP ultrawide",
				"memory":               "8 GB",
				"storageCapacity":      "256 GB",
				"brand":                "Atlas",
				"modelVersion":         "One 2026",
				"operatingSystem":      "Android 16",
			},
		},
		{
			ID:          "phone-2",
			Name:        "Nimbus Pro",
			ImageURL:    "https://example.com/images/nimbus-pro.png",
			Description: "High-end smartphone with strong video capture and storage.",
			Price:       4599.00,
			Rating:      4.8,
			Size:        "158.4 x 72.0 x 7.6 mm",
			Weight:      "181 g",
			Color:       "Silver",
			Specifications: map[string]string{
				"batteryCapacity":      "4700 mAh",
				"cameraSpecifications": "48 MP wide + 48 MP telephoto + 12 MP ultrawide",
				"memory":               "12 GB",
				"storageCapacity":      "512 GB",
				"brand":                "Nimbus",
				"modelVersion":         "Pro X",
				"operatingSystem":      "Android 16",
			},
		},
		{
			ID:          "speaker-1",
			Name:        "Pulse Mini",
			ImageURL:    "https://example.com/images/pulse-mini.png",
			Description: "Compact speaker for desk setups and small rooms.",
			Price:       499.90,
			Rating:      4.4,
			Size:        "98 x 98 x 120 mm",
			Weight:      "740 g",
			Color:       "Blue",
			Specifications: map[string]string{
				"brand":           "Pulse",
				"batteryCapacity": "12 hours",
				"connectivity":    "Bluetooth 5.3, USB-C",
			},
		},
		{
			ID:          "phone-3",
			Name:        "Orion Lite",
			ImageURL:    "https://example.com/images/orion-lite.png",
			Description: "Affordable smartphone with essential features.",
			Price:       1899.00,
			Rating:      4.2,
			Size:        "150.9 x 75.7 x 8.3 mm",
			Weight:      "175 g",
			Color:       "Black",
			Specifications: map[string]string{
				"batteryCapacity":      "4000 mAh",
				"cameraSpecifications": "13 MP wide",
				"memory":               "4 GB",
				"storageCapacity":      "64 GB",
				"brand":                "Orion",
				"modelVersion":         "Lite 2026",
				"operatingSystem":      "Android 12",
			},
		},
		{
			ID:          "microwave-1",
			Name:        "HeatMaster 25L",
			ImageURL:    "https://example.com/images/heatmaster-25l.png",
			Description: "Compact microwave oven with 25L capacity.",
			Price:       899.00,
			Rating:      4.5,
			Size:        "485 x 395 x 292 mm",
			Weight:      "12.5 kg",
			Color:       "White",
			Specifications: map[string]string{
				"capacity":     "25 L",
				"power":        "900 W",
				"brand":        "HeatMaster",
				"modelVersion": "25L Compact",
			},
		},
	}
}
