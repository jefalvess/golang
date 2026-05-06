package model

type Product struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	ImageURL       string            `json:"imageUrl"`
	Description    string            `json:"description"`
	Price          float64           `json:"price"`
	Rating         float64           `json:"rating"`
	Size           string            `json:"size"`
	Weight         string            `json:"weight"`
	Color          string            `json:"color"`
	Specifications map[string]string `json:"specifications,omitempty"`
	Type           string            `json:"type"`
	SpecsTable     string            `json:"-"` // nome da tabela de specs — uso interno apenas
}

func (p Product) FieldMap() map[string]any {
	return map[string]any{
		"id":             p.ID,
		"name":           p.Name,
		"imageUrl":       p.ImageURL,
		"description":    p.Description,
		"price":          p.Price,
		"rating":         p.Rating,
		"size":           p.Size,
		"weight":         p.Weight,
		"color":          p.Color,
		"specifications": p.Specifications,
		"type":           p.Type,
	}
}
