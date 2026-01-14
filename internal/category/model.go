package category

type Category struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Subcategories []*Subcategory `json:"subcategories"`
}

type Subcategory struct {
	ID         string `json:"id"`
	CategoryID string `json:"categoryID"`
	Name       string `json:"name"`
}
