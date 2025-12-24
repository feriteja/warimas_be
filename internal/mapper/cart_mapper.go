package mapper

// import (
// 	"fmt"
// 	"time"
// 	"warimas-be/internal/cart"
// 	"warimas-be/internal/graph/model"
// 	"warimas-be/internal/product"
// )

// func MapVariantToGQL(v product.Variant) *model.Variant {
// 	var imgURL string
// 	if v.ImageUrl != nil {
// 		imgURL = *v.ImageUrl
// 	}

// 	return &model.Variant{
// 		ID:           v.ID,
// 		Name:         v.Name,
// 		ProductID:    v.ProductID,
// 		QuantityType: v.QuantityType,
// 		Price:        v.Price,
// 		Stock:        int32(v.Stock),
// 		ImageURL:     imgURL,
// 	}
// }

// func MapVariantsToGQL(vars []*product.Variant) []*model.Variant {
// 	res := make([]*model.Variant, 0, len(vars))
// 	for _, v := range vars {
// 		res = append(res, MapVariantToGQL(*v))
// 	}
// 	return res
// }

// func MapProductToGQL(p product.Product) *model.ProductCart {
// 	return &model.ProductCart{
// 		ID:      fmt.Sprint(p.ID),
// 		Name:    p.Name,
// 		Variant: MapVariantsToGQL(p.Variants),
// 	}
// }

// func MapCartItemToGQL(ci cart.CartItem) *model.CartItem {
// 	return &model.CartItem{
// 		ID:        fmt.Sprint(ci.ID),
// 		UserID:    fmt.Sprint(ci.UserID),
// 		Product:   MapProductToGQL(ci.Product),
// 		CreatedAt: ci.CreatedAt.Format(time.RFC3339),
// 		UpdatedAt: ci.UpdatedAt.Format(time.RFC3339),
// 	}
// }
