package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/graph/model"
	servicepkg "warimas-be/internal/service"
)

type Repository interface {
	GetAll(opts servicepkg.ProductQueryOptions) ([]model.CategoryProduct, error)
	Create(p model.Product) (model.Product, error)
	BulkCreateVariants(
		ctx context.Context,
		input []*model.NewVariant,
		sellerID string,
	) ([]*model.Variant, error)
	GetPackages(ctx context.Context, filter *model.PackageFilterInput, sort *model.PackageSortInput, limit, offset int32) ([]*model.Package, error)
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

func (r *repository) BulkCreateVariants(
	ctx context.Context,
	input []*model.NewVariant,
	sellerID string,
) ([]*model.Variant, error) {

	if len(input) > 100 {
		return nil, errors.New("max 100 variants per request")
	}

	query := `
		INSERT INTO variants (
			product_id,
			name,
			quantity_type,
			price,
			stock,
			imageurl,
			subcategory_id
		) VALUES
	`

	args := []interface{}{}
	valueStrings := []string{}

	for i, v := range input {
		idx := i * 7

		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				idx+1, idx+2, idx+3, idx+4,
				idx+5, idx+6, idx+7,
			),
		)

		args = append(args,
			v.ProductID,
			v.Name,
			v.QuantityType,
			v.Price,
			v.Stock,
			v.ImageURL,
			v.SubcategoryID,
		)
	}

	query += strings.Join(valueStrings, ",")
	query += `
		RETURNING 
			id,
			product_id,
			name,
			quantity_type,
			price,
			stock,
			imageurl,
			subcategory_id,
			created_at
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []*model.Variant

	for rows.Next() {
		var v model.Variant
		err := rows.Scan(
			&v.ID,
			&v.ProductID,
			&v.Name,
			&v.QuantityType,
			&v.Price,
			&v.Stock,
			&v.ImageURL,
			&v.SubcategoryID,
			&v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		variants = append(variants, &v)
	}

	return variants, nil
}

func (r *repository) GetPackages(
	ctx context.Context,
	filter *model.PackageFilterInput,
	sort *model.PackageSortInput,
	limit, offset int32,
) ([]*model.Package, error) {

	query := `
		SELECT
			p.id,
			p.name,
			p.image_url,
			p.user_id,
			pi.id AS item_id,
			pi.variant_id,
			pi.name AS item_name,
			pi.image_url AS item_image_url,
			v.price AS variant_price,
			pi.quantity,
			pi.created_at,
			pi.updated_at
		FROM packages p
		LEFT JOIN package_items pi ON p.id = pi.package_id
		LEFT JOIN variants v ON pi.variant_id = v.id
		WHERE 1=1
	`

	// Filtering
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.ID != nil {
			query += fmt.Sprintf(" AND p.id = $%d", argIndex)
			args = append(args, *filter.ID)
			argIndex++
		}
		if filter.Name != nil && *filter.Name != "" {
			query += fmt.Sprintf(" AND p.name ILIKE $%d", argIndex)
			args = append(args, "%"+*filter.Name+"%")
			argIndex++
		}
	}

	// Sorting
	if sort != nil {
		switch sort.Field {
		case model.PackageSortFieldName:
			query += " ORDER BY p.name " + strings.ToUpper(string(sort.Direction))
		case model.PackageSortFieldCreatedAt:
			query += " ORDER BY p.created_at " + strings.ToUpper(string(sort.Direction))
		default:
			query += " ORDER BY p.created_at DESC"
		}
	} else {
		query += " ORDER BY p.created_at DESC"
	}

	// Pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packagesMap := map[string]*model.Package{}

	for rows.Next() {
		var (
			p            model.Package
			pi           model.PackageItem
			variantPrice sql.NullFloat64
			imageURL     sql.NullString
			userID       sql.NullString
		)

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&imageURL,
			&userID,
			&pi.ID,
			&pi.VariantID,
			&pi.Name,
			&pi.ImageURL,
			&variantPrice,
			&pi.Quantity,
			&pi.CreatedAt,
			&pi.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if imageURL.Valid {
			p.ImageURL = &imageURL.String
		}
		if userID.Valid {
			p.UserID = &userID.String
		}

		// check if package exists already
		if _, ok := packagesMap[p.ID]; !ok {
			p.Items = []*model.PackageItem{}
			packagesMap[p.ID] = &p
		}

		// If no item exists (NULL), skip append
		if pi.ID != "" {
			pi.Price = variantPrice.Float64
			packagesMap[p.ID].Items = append(packagesMap[p.ID].Items, &pi)
		}
	}

	// Convert map to slice
	result := []*model.Package{}
	for _, pkg := range packagesMap {
		result = append(result, pkg)
	}

	return result, nil
}
