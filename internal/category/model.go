package category

type Category struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	Subcategories []*Subcategory `json:"subcategories"`
}

type Subcategory struct {
	ID         string `json:"id"`
	CategoryID string `json:"categoryID"`
	Name       string `json:"name"`
}
