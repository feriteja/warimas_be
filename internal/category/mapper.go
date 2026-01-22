package category

import (
	"warimas-be/internal/graph/model"
)

func MapCategoryToGraphQL(c *Category) *model.Category {
	if c == nil {
		return nil
	}

	items := make([]*model.Subcategory, 0, len(c.Subcategories))
	for _, item := range c.Subcategories {
		items = append(items, MapSubcategoriesToGraphQL(item))
	}

	return &model.Category{
		ID:            c.ID,
		Name:          c.Name,
		Slug:          c.Slug,
		Subcategories: items,
	}
}

func MapSubcategoriesToGraphQL(sc *Subcategory) *model.Subcategory {
	return &model.Subcategory{
		ID:         sc.ID,
		CategoryID: sc.CategoryID,
		Name:       sc.Name,
	}
}
