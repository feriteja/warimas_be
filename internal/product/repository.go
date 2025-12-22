package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"warimas-be/internal/graph/model"
	servicepkg "warimas-be/internal/service"
	"warimas-be/internal/utils"
)

type Repository interface {
	GetProductsByGroup(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.ProductByCategory, error)
	GetList(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]*model.Product, error)
	Create(ctx context.Context, input model.NewProduct, sellerID string) (model.Product, error)
	Update(ctx context.Context, input model.UpdateProduct, sellerID string) (model.Product, error)
	BulkCreateVariants(
		ctx context.Context,
		input []*model.NewVariant,
		sellerID string,
	) ([]*model.Variant, error)
	BulkUpdateVariants(
		ctx context.Context,
		input []*model.UpdateVariant,
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

func (r *repository) GetProductsByGroup(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.ProductByCategory, error) {

	query := `
	SELECT
	    c.id AS category_id,
     c.name AS category_name,
     s.id AS subcategory_id,
     s.name AS subcategory_name,
    p_total.total_products,

    p.id AS product_id,
    p.name AS product_name,
    p.seller_id,
    p.slug,

    v.id AS variant_id,
	v.product_id AS variant_product_id,
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
LEFT JOIN subcategories s ON s.id = p.subcategory_id


ORDER BY c.name, p.name, v.name;

`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]*model.ProductByCategory)
	productMap := make(map[string]*model.Product)
	categoryOrder := []string{}
	productCount := make(map[string]int) // track product count
	const limitPerCategory = 10

	for rows.Next() {

		var (
			categoryId      sql.NullString
			categoryName    sql.NullString
			subcategoryId   sql.NullString
			subcategoryName sql.NullString
			totalProducts   sql.NullInt32

			pID       sql.NullString
			pName     sql.NullString
			pSellerID sql.NullString
			pSlug     sql.NullString

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
			&subcategoryId,
			&subcategoryName,
			&totalProducts,

			&pID, &pName, &pSellerID, &pSlug,
			&vID, &vProdID, &vName, &vPrice, &vStock, &vImageURL,
		); err != nil {
			return nil, err
		}

		cat := categoryId.String

		//--------------------------------------
		// CATEGORY
		//--------------------------------------
		if _, exists := categoryMap[cat]; !exists {
			categoryMap[cat] = &model.ProductByCategory{
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
					ID:              pID.String,
					Name:            pName.String,
					SellerID:        pSellerID.String,
					CategoryID:      categoryId.String,
					CategoryName:    categoryName.String,
					SubcategoryID:   subcategoryId.String,
					SubcategoryName: subcategoryName.String,
					Slug:            pSlug.String,
					Variants:        []*model.Variant{},
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
	result := make([]model.ProductByCategory, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		result = append(result, *categoryMap[cat])
	}

	return result, nil
}

func (r *repository) GetList(
	ctx context.Context,
	opts servicepkg.ProductQueryOptions,
) ([]*model.Product, error) {

	query := `
SELECT
    p.id,
    p.name,
    p.seller_id,
    p.category_id,
    p.subcategory_id,
    p.slug,
    p.imageurl,
    p.description,
    p.created_at,

    c.name AS category_name,
    s.name AS subcategory_name,

    COALESCE(
        json_agg(
            json_build_object(
                'id', v.id,
                'productId', v.product_id,
                'name', v.name,
                'price', v.price,
                'stock', v.stock,
                'imageUrl', v.imageurl
            )
        ) FILTER (WHERE v.id IS NOT NULL),
        '[]'
    ) AS variants

FROM products p
LEFT JOIN category c ON c.id = p.category_id
LEFT JOIN subcategories s ON s.id = p.subcategory_id
LEFT JOIN variants v ON v.product_id = p.id
`

	var (
		args  []any
		where []string
	)

	/* ---------------- FILTERS ---------------- */

	if opts.Filter != nil {

		if opts.Filter.Category != nil {
			args = append(args, *opts.Filter.Category)
			where = append(where, fmt.Sprintf("p.category_id = $%d", len(args)))
		}

		if opts.Filter.Search != nil {
			args = append(args, "%"+*opts.Filter.Search+"%")
			where = append(where, fmt.Sprintf("p.name ILIKE $%d", len(args)))
		}

		if opts.Filter.MinPrice != nil {
			args = append(args, *opts.Filter.MinPrice)
			where = append(where, fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM variants v2
					WHERE v2.product_id = p.id
					AND v2.price >= $%d
				)
			`, len(args)))
		}

		if opts.Filter.MaxPrice != nil {
			args = append(args, *opts.Filter.MaxPrice)
			where = append(where, fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM variants v2
					WHERE v2.product_id = p.id
					AND v2.price <= $%d
				)
			`, len(args)))
		}

		if opts.Filter.InStock != nil && *opts.Filter.InStock {
			where = append(where, `
				EXISTS (
					SELECT 1 FROM variants v2
					WHERE v2.product_id = p.id
					AND v2.stock > 0
				)
			`)
		}
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	/* ---------------- GROUP BY ---------------- */

	query += `
		GROUP BY
			p.id,
			p.name,
			p.seller_id,
			p.category_id,
			p.subcategory_id,
			p.slug,
			p.imageurl,
			p.description,
			p.created_at,
			c.name,
			s.name
`

	/* ---------------- SORTING ---------------- */

	if opts.Sort != nil {
		dir := "ASC"
		if opts.Sort.Direction == model.SortDirectionDesc {
			dir = "DESC"
		}

		switch opts.Sort.Field {
		case model.ProductSortFieldPrice:
			query += fmt.Sprintf(" ORDER BY MIN(v.price) %s", dir)
		case model.ProductSortFieldName:
			query += fmt.Sprintf(" ORDER BY p.name %s", dir)
		default:
			query += " ORDER BY p.created_at DESC"
		}
	} else {
		query += " ORDER BY p.created_at DESC"
	}

	/* ---------------- PAGINATION ---------------- */

	if opts.Limit != nil {
		args = append(args, *opts.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	if opts.Offset != nil {
		args = append(args, *opts.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	/* ---------------- EXECUTION ---------------- */

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product

	for rows.Next() {
		var (
			p            model.Product
			variantsJSON []byte
		)

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.SellerID,
			&p.CategoryID,
			&p.SubcategoryID, // ✅ NULL-safe
			&p.Slug,
			&p.ImageURL,
			&p.Description,
			&p.CreatedAt,
			&p.CategoryName,
			&p.SubcategoryName,
			&variantsJSON,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(variantsJSON, &p.Variants); err != nil {
			return nil, err
		}

		products = append(products, &p)
	}

	return products, nil
}

func (r *repository) Create(ctx context.Context, input model.NewProduct, sellerID string) (model.Product, error) {

	var p model.Product

	if sellerID == "" {
		return p, errors.New("sellerID is required")
	}

	if input.CategoryID == "" {
		return p, errors.New("categoryID is required")
	}

	log.Println("sellerID:", sellerID)
	log.Println("categoryID:", input.CategoryID)

	err := r.db.QueryRow(
		"INSERT INTO products (category_id, seller_id, name, slug, imageurl) VALUES ($1, $2, $3, $4, $5) RETURNING id, name, imageurl",
		input.CategoryID, sellerID, input.Name, utils.Slugify(input.Name, sellerID), input.ImageURL,
	).Scan(&p.ID, &p.Name, &p.ImageURL)
	return p, err
}

func (r *repository) Update(
	ctx context.Context,
	input model.UpdateProduct,
	sellerID string,
) (model.Product, error) {

	query := `
		UPDATE products
		SET %s
		WHERE id = $%d AND seller_id = $%d
		RETURNING id, name, imageurl, description, category_id, seller_id
	`

	setClauses := []string{}
	args := []any{}
	argPos := 1

	if input.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d, slug = $%d", argPos, argPos+1))
		args = append(args, *input.Name, utils.Slugify(*input.Name, sellerID))
		argPos += 2
	}

	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *input.Status)
		argPos++
	}

	if input.ImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("imageurl = $%d", argPos))
		args = append(args, *input.ImageURL)
		argPos++
	}

	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *input.Description)
		argPos++
	}

	if input.CategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("category_id = $%d", argPos))
		args = append(args, *input.CategoryID)
		argPos++
	}

	// WHERE clause args
	args = append(args, input.ID, sellerID)

	finalQuery := fmt.Sprintf(
		query,
		strings.Join(setClauses, ", "),
		argPos,
		argPos+1,
	)

	var product model.Product
	err := r.db.QueryRowContext(ctx, finalQuery, args...).Scan(
		&product.ID,
		&product.Name,
		&product.ImageURL,
		&product.Description,
		&product.CategoryID,
		&product.SellerID,
	)

	if err == sql.ErrNoRows {
		return model.Product{}, errors.New("product not found or not owned by seller")
	}

	return product, err
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
			description
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
			v.Description,
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
			&v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		variants = append(variants, &v)
	}

	return variants, nil
}

func (r *repository) BulkUpdateVariants(
	ctx context.Context,
	input []*model.UpdateVariant,
	sellerID string,
) ([]*model.Variant, error) {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var updatedVariants []*model.Variant

	for _, v := range input {
		setClauses := []string{}
		args := []any{}
		argPos := 1

		if v.QuantityType != nil {
			setClauses = append(setClauses, fmt.Sprintf("quantity_type = $%d", argPos))
			args = append(args, *v.QuantityType)
			argPos++
		}

		if v.Name != nil {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argPos))
			args = append(args, *v.Name)
			argPos++
		}

		if v.Price != nil {
			setClauses = append(setClauses, fmt.Sprintf("price = $%d", argPos))
			args = append(args, *v.Price)
			argPos++
		}

		if v.Stock != nil {
			setClauses = append(setClauses, fmt.Sprintf("stock = $%d", argPos))
			args = append(args, *v.Stock)
			argPos++
		}

		if v.ImageURL != nil {
			setClauses = append(setClauses, fmt.Sprintf("imageurl = $%d", argPos))
			args = append(args, *v.ImageURL)
			argPos++
		}

		if v.Description != nil {
			setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
			args = append(args, *v.Description)
			argPos++
		}

		// WHERE args
		args = append(args, v.ID, v.ProductID, sellerID)

		query := fmt.Sprintf(`
			UPDATE variants
			SET %s
			WHERE id = $%d
			  AND product_id = $%d
			  AND product_id IN (
			    SELECT id FROM products WHERE seller_id = $%d
			  )
			RETURNING id, product_id, name, price, stock, imageurl, description
		`,
			strings.Join(setClauses, ", "),
			argPos,
			argPos+1,
			argPos+2,
		)

		var variant model.Variant
		err := tx.QueryRowContext(ctx, query, args...).Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.Name,
			&variant.Price,
			&variant.Stock,
			&variant.ImageURL,
			&variant.Description,
		)
		if err != nil {
			return nil, err
		}

		updatedVariants = append(updatedVariants, &variant)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return updatedVariants, nil
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
