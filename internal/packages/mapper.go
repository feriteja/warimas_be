package packages

import (
	"warimas-be/internal/graph/model"
)

func MapPackageToGraphQL(p *Package) *model.Package {
	items := make([]*model.PackageItem, len(p.Items))
	for i, item := range p.Items {
		items[i] = &model.PackageItem{
			ID:        item.ID,
			PackageID: item.PackageID,
			VariantID: item.VariantID,
			ImageURL:  item.ImageURL,
			Name:      item.Name,
			Price:     item.Price,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}

	var userID int32
	if p.UserID != nil {
		userID = int32(*p.UserID)
	}

	return &model.Package{
		ID:        p.ID,
		Name:      p.Name,
		ImageURL:  p.ImageURL,
		UserID:    &userID,
		Items:     items,
		Type:      p.Type,
		IsActive:  true,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
