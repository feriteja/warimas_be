package product

import (
	"database/sql"
	"warimas-be/internal/graph/model"
	servicepkg "warimas-be/internal/service"
)

type Repository interface {
	GetAll(opts servicepkg.ProductQueryOptions) ([]model.CategoryProduct, error)
	Create(p model.Product) (model.Product, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetAll(opts servicepkg.ProductQueryOptions) ([]model.CategoryProduct, error) {

	query := `SELECT
	    c.id AS category_id,
     c.name AS category_name,
    p_total.total_products,

    p.id AS product_id,
    p.name AS product_name,
    p.seller_id,
    p.slug,
    p.price,

    v.id AS variant_id,
    v.name AS variant_name,
    v.price AS variant_price,
    v.stock,
    v.imageurl

FROM category c

-- Total products per category
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS total_products
    FROM products p
    WHERE p.category_id = c.id
) AS p_total ON true

-- Limit 10 products per category
LEFT JOIN LATERAL (
    SELECT *
    FROM products p
    WHERE p.category_id = c.id
    ORDER BY p.name
    LIMIT 10
) AS p ON true

-- Join variants
LEFT JOIN variants v ON v.product_id = p.id

ORDER BY c.name, p.name, v.name;

`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]*model.CategoryProduct)
	productMap := make(map[string]*model.Product)
	categoryOrder := []string{}
	productCount := make(map[string]int) // track product count
	const limitPerCategory = 10

	for rows.Next() {

		var (
			categoryId    sql.NullString
			categoryName  sql.NullString
			totalProducts sql.NullInt32

			pID       sql.NullString
			pName     sql.NullString
			pSellerID sql.NullString
			pSlug     sql.NullString
			pPrice    sql.NullFloat64

			vID       sql.NullString
			vProdID   sql.NullString
			vName     sql.NullString
			vPrice    sql.NullFloat64
			vStock    sql.NullInt32
			vImageURL sql.NullString
		)

		if err := rows.Scan(
			&categoryId,
			&categoryName,
			&totalProducts,

			&pID, &pName, &pSellerID, &pSlug, &pPrice,
			&vID, &vName, &vPrice, &vStock, &vImageURL,
		); err != nil {
			return nil, err
		}

		cat := categoryName.String

		//--------------------------------------
		// CATEGORY
		//--------------------------------------
		if _, exists := categoryMap[cat]; !exists {
			categoryMap[cat] = &model.CategoryProduct{
				CategoryName:  &cat,
				TotalProducts: int32(totalProducts.Int32),
				Products:      []*model.Product{},
			}
			categoryOrder = append(categoryOrder, cat)
			productCount[cat] = 0
		}

		//--------------------------------------
		// PRODUCT (limit to 10)
		//--------------------------------------
		if pID.Valid {

			// Skip if we already reached 10 products
			if productCount[cat] >= limitPerCategory {
				continue
			}

			if _, exists := productMap[pID.String]; !exists {
				productMap[pID.String] = &model.Product{
					ID:         pID.String,
					Name:       pName.String,
					SellerID:   pSellerID.String,
					CategoryID: &categoryId.String,
					Slug:       pSlug.String,
					Price:      pPrice.Float64,
					Variants:   []*model.Variant{},
				}

				categoryMap[cat].Products = append(categoryMap[cat].Products, productMap[pID.String])
				productCount[cat]++
			}
		}

		//--------------------------------------
		// VARIANT (only add if product included)
		//--------------------------------------
		if vID.Valid {
			if prod, ok := productMap[pID.String]; ok {
				prod.Variants = append(prod.Variants, &model.Variant{
					ID:        vID.String,
					ProductID: vProdID.String,
					Name:      vName.String,
					Price:     vPrice.Float64,
					Stock:     vStock.Int32,
					ImageURL:  vImageURL.String,
				})
			}
		}
	}

	//--------------------------------------
	// Convert to ordered slice
	//--------------------------------------
	result := make([]model.CategoryProduct, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		result = append(result, *categoryMap[cat])
	}

	return result, nil
}

func (r *repository) Create(p model.Product) (model.Product, error) {
	err := r.db.QueryRow(
		"INSERT INTO products (name, price, stock) VALUES ($1, $2, $3) RETURNING id",
		p.Name, p.Price, p.Stock,
	).Scan(&p.ID)
	return p, err
}
