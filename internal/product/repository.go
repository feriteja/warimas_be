package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

type Repository interface {
	GetProductsByGroup(ctx context.Context, opts ProductQueryOptions) ([]ProductByCategory, error)
	GetList(ctx context.Context, opts ProductQueryOptions) ([]*Product, *int, error)
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
	GetPackages(ctx context.Context, filter *model.PackageFilterInput, sort *model.PackageSortInput, limit, page int32, includeDisabled bool) ([]*model.Package, error)
	GetProductByID(ctx context.Context, productParams GetProductOptions) (*Product, error)
	GetProductVariantByID(ctx context.Context, productParams GetVariantOptions) (*Variant, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

var ErrRepositoryFailure = errors.New("internal data access error")

func (r *repository) GetProductsByGroup(
	ctx context.Context,
	opts ProductQueryOptions,
) ([]ProductByCategory, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetProductsByGroup"),
	)

	start := time.Now()
	log.Info("start get products by group")
	log.Debug("query options", zap.Any("opts", opts))

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

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		log.Error("failed to query products by group", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]*ProductByCategory)
	productMap := make(map[string]*Product)
	categoryOrder := make([]string, 0)
	const limitPerCategory = 10

	for rows.Next() {

		var (
			categoryID      sql.NullString
			categoryName    sql.NullString
			subcategoryID   sql.NullString
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
			&categoryID,
			&categoryName,
			&subcategoryID,
			&subcategoryName,
			&totalProducts,
			&pID, &pName, &pSellerID, &pSlug,
			&vID, &vProdID, &vName, &vPrice, &vStock, &vImageURL,
		); err != nil {
			log.Error("failed to scan grouped product row", zap.Error(err))
			return nil, err
		}

		if !categoryID.Valid {
			// Should never happen, but stay defensive
			continue
		}

		catID := categoryID.String

		//--------------------------------------
		// CATEGORY
		//--------------------------------------
		if _, ok := categoryMap[catID]; !ok {
			categoryMap[catID] = &ProductByCategory{
				CategoryName:  categoryName.String,
				TotalProducts: int(totalProducts.Int32),
				Products:      make([]*Product, 0, limitPerCategory),
			}
			categoryOrder = append(categoryOrder, catID)
		}

		//--------------------------------------
		// PRODUCT (limit to 10 per category)
		//--------------------------------------
		if pID.Valid {
			productKey := catID + ":" + pID.String

			if _, ok := productMap[productKey]; !ok {
				product := &Product{
					ID:              pID.String,
					Name:            pName.String,
					SellerID:        pSellerID.String,
					CategoryID:      catID,
					CategoryName:    categoryName.String,
					SubcategoryID:   subcategoryID.String,
					SubcategoryName: subcategoryName.String,
					Slug:            pSlug.String,
					Variants:        make([]*Variant, 0, 4),
				}

				productMap[productKey] = product
				categoryMap[catID].Products = append(categoryMap[catID].Products, product)
			}

			//--------------------------------------
			// VARIANT
			//--------------------------------------
			if vID.Valid {
				productMap[productKey].Variants = append(
					productMap[productKey].Variants,
					&Variant{
						ID:        vID.String,
						ProductID: vProdID.String,
						Name:      vName.String,
						Price:     vPrice.Float64,
						Stock:     vStock.Int32,
						ImageURL:  vImageURL.String,
					},
				)
			}
		}
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	//--------------------------------------
	// Convert to ordered slice
	//--------------------------------------
	result := make([]ProductByCategory, 0, len(categoryOrder))
	for _, id := range categoryOrder {
		result = append(result, *categoryMap[id])
	}

	log.Info("success get products by group",
		zap.Int("category_count", len(result)),
		zap.Duration("duration", time.Since(start)),
	)

	return result, nil
}

func (r *repository) GetList(
	ctx context.Context,
	opts ProductQueryOptions,
) ([]*Product, *int, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetProductList"),
	)

	start := time.Now()

	baseQuery := `
		FROM products p
		LEFT JOIN sellers ON sellers.id = p.seller_id
		LEFT JOIN category c ON c.id = p.category_id
		LEFT JOIN subcategories s ON s.id = p.subcategory_id
		LEFT JOIN variants v ON v.product_id = p.id
	`

	var (
		args   []any
		where  []string
		having []string
	)

	var totalProduct *int

	/* ---------- FILTERS ---------- */

	if opts.SellerID != nil {
		args = append(args, *opts.SellerID)
		where = append(where, fmt.Sprintf("p.seller_id = $%d", len(args)))
	}

	if opts.CategoryID != nil {
		args = append(args, *opts.CategoryID)
		where = append(where, fmt.Sprintf("p.category_id = $%d", len(args)))
	}

	if opts.SellerName != nil {
		args = append(args, "%"+*opts.SellerName+"%")
		where = append(where, fmt.Sprintf("sellers.name ILIKE $%d", len(args)))
	}

	if opts.Search != nil {
		args = append(args, "%"+*opts.Search+"%")
		where = append(where, fmt.Sprintf("p.name ILIKE $%d", len(args)))
	}

	if opts.InStock != nil && *opts.InStock {
		where = append(where, `
			EXISTS (
				SELECT 1 FROM variants v2
				WHERE v2.product_id = p.id
				AND v2.stock > 0
			)
		`)
	}

	// ---- STATUS & VISIBILITY (single source of truth) ----
	if opts.Status != nil {
		args = append(args, *opts.Status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	} else if opts.OnlyActive {
		where = append(where, "p.status = 'active'")
	}

	/* ---------- PRICE FILTERS (HAVING) ---------- */

	if opts.MinPrice != nil {
		args = append(args, *opts.MinPrice)
		having = append(
			having,
			fmt.Sprintf("MIN(v.price) IS NOT NULL AND MIN(v.price) >= $%d", len(args)),
		)
	}

	if opts.MaxPrice != nil {
		args = append(args, *opts.MaxPrice)
		having = append(
			having,
			fmt.Sprintf("MIN(v.price) IS NOT NULL AND MIN(v.price) <= $%d", len(args)),
		)
	}

	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}

	/* ---------- PAGINATION NORMALIZATION ---------- */

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	page := opts.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	/* ---------- DEBUG INPUT LOG ---------- */

	log.Debug("get product list started",
		zap.Int32("page", page),
		zap.Int32("limit", limit),
		zap.Int("where_conditions", len(where)),
		zap.Int("having_conditions", len(having)),
		zap.Bool("include_count", opts.IncludeCount),
	)

	/* ---------- COUNT QUERY ---------- */

	if opts.IncludeCount {
		countQuery := `
			SELECT COUNT(*) FROM (
				SELECT p.id
		` + baseQuery + `
				GROUP BY p.id
		`

		if len(having) > 0 {
			countQuery += " HAVING " + strings.Join(having, " AND ")
		}

		countQuery += ") AS sub"

		var total int
		if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			log.Error("count query failed", zap.Error(err))
			return nil, nil, err
		}
		totalProduct = &total
	}

	/* ---------- DATA QUERY ---------- */

	selectQuery := `
SELECT
	p.id,
	p.name,
	p.seller_id,
	COALESCE(sellers.name, 'Unknown') AS seller_name,
	p.status,
	p.category_id,
	p.subcategory_id,
	p.slug,
	p.imageurl,
	p.description,
	p.created_at,
	p.updated_at,
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
` + baseQuery + `
GROUP BY
	p.id, sellers.name, c.name, s.name
`

	if len(having) > 0 {
		selectQuery += " HAVING " + strings.Join(having, " AND ")
	}

	/* ---------- SORT ---------- */

	orderBy := "p.created_at"
	switch opts.SortField {
	case ProductSortFieldPrice:
		orderBy = "MIN(v.price)"
	case ProductSortFieldName:
		orderBy = "p.name"
	}

	dir := "DESC"
	if opts.SortDirection == SortDirectionAsc {
		dir = "ASC"
	}

	selectQuery += " ORDER BY " + orderBy + " " + dir

	/* ---------- LIMIT / OFFSET ---------- */

	args = append(args, limit, offset)
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	/* ---------- EXEC ---------- */

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		log.Error("data query failed", zap.Error(err))
		return nil, totalProduct, err
	}
	defer rows.Close()

	var products []*Product

	for rows.Next() {
		var (
			p            Product
			variantsJSON []byte
		)

		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.SellerID,
			&p.SellerName,
			&p.Status,
			&p.CategoryID,
			&p.SubcategoryID,
			&p.Slug,
			&p.ImageURL,
			&p.Description,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.CategoryName,
			&p.SubcategoryName,
			&variantsJSON,
		); err != nil {
			log.Error("row scan failed", zap.Error(err))
			return nil, totalProduct, err
		}

		if err := json.Unmarshal(variantsJSON, &p.Variants); err != nil {
			log.Warn("failed to unmarshal variants",
				zap.String("product_id", p.ID),
				zap.Error(err),
			)
		}

		products = append(products, &p)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration failed", zap.Error(err))
		return nil, totalProduct, ErrRepositoryFailure
	}

	/* ---------- SUCCESS LOG ---------- */

	fields := []zap.Field{
		zap.Int("count", len(products)),
		zap.Int32("page", page),
		zap.Int32("limit", limit),
		zap.Duration("duration", time.Since(start)),
	}

	if totalProduct != nil {
		fields = append(fields, zap.Int("total", *totalProduct))
	}

	log.Info("get product list success", fields...)

	return products, totalProduct, nil
}

func (r *repository) Create(
	ctx context.Context,
	input model.NewProduct,
	sellerID string,
) (model.Product, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "CreateProduct"),
		zap.String("seller_id", sellerID),
		zap.String("category_id", input.CategoryID),
		zap.String("subcategory_id", input.SubcategoryID),
	)

	start := time.Now()
	log.Info("start create product")

	var p model.Product

	// 🔒 Validation
	if sellerID == "" {
		log.Warn("create product failed: missing sellerID")
		return p, errors.New("sellerID is required")
	}

	if input.Name == "" {
		log.Warn("create product failed: missing name")
		return p, errors.New("name is required")
	}

	if input.CategoryID == "" {
		log.Warn("create product failed: missing categoryID")
		return p, errors.New("categoryID is required")
	}

	if input.SubcategoryID == "" {
		log.Warn("create product failed: missing subcategoryID")
		return p, errors.New("subcategoryID is required")
	}

	slug := utils.Slugify(input.Name, sellerID)

	err := r.db.QueryRowContext(
		ctx,
		`
		INSERT INTO products (
			category_id,
			seller_id,
			name,
			slug,
			imageurl,
			subcategory_id,
			description
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, imageurl
		`,
		input.CategoryID,
		sellerID,
		input.Name,
		slug,
		input.ImageURL,
		input.SubcategoryID,
		input.Description,
	).Scan(
		&p.ID,
		&p.Name,
		&p.ImageURL,
	)

	if err != nil {
		log.Error("failed to create product", zap.Error(err))
		return p, err
	}

	log.Info("success create product",
		zap.String("product_id", p.ID),
		zap.Duration("duration", time.Since(start)),
	)

	return p, nil
}

func (r *repository) Update(
	ctx context.Context,
	input model.UpdateProduct,
	sellerID string,
) (model.Product, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "UpdateProduct"),
		zap.String("product_id", input.ID),
		zap.String("seller_id", sellerID),
	)

	start := time.Now()
	log.Info("start update product")

	queryTpl := `
		UPDATE products
		SET %s
		WHERE id = $%d AND seller_id = $%d
		RETURNING id, name, imageurl, description, category_id, seller_id, subcategory_id, status
	`

	setClauses := make([]string, 0, 6)
	args := make([]any, 0, 8)
	updatedFields := make([]string, 0, 6)
	argPos := 1

	if input.Name != nil {
		setClauses = append(setClauses,
			fmt.Sprintf("name = $%d, slug = $%d", argPos, argPos+1),
		)
		args = append(args, *input.Name, utils.Slugify(*input.Name, sellerID))
		updatedFields = append(updatedFields, "name", "slug")
		argPos += 2
	}

	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *input.Status)
		updatedFields = append(updatedFields, "status")
		argPos++
	}

	if input.ImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("imageurl = $%d", argPos))
		args = append(args, *input.ImageURL)
		updatedFields = append(updatedFields, "imageurl")
		argPos++
	}

	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *input.Description)
		updatedFields = append(updatedFields, "description")
		argPos++
	}

	if input.CategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("category_id = $%d", argPos))
		args = append(args, *input.CategoryID)
		updatedFields = append(updatedFields, "category_id")
		argPos++
	}

	if input.SubcategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("subcategory_id = $%d", argPos))
		args = append(args, *input.SubcategoryID)
		updatedFields = append(updatedFields, "subcategory_id")
		argPos++
	}

	// 🚨 No fields to update
	if len(setClauses) == 0 {
		log.Warn("update product skipped: no fields to update")
		return model.Product{}, errors.New("no fields to update")
	}

	// WHERE clause args
	args = append(args, input.ID, sellerID)

	finalQuery := fmt.Sprintf(
		queryTpl,
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
		&product.SubcategoryID,
		&product.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Warn("product not found or not owned by seller")
			return model.Product{}, errors.New("product not found or not owned by seller")
		}

		log.Error("failed to update product", zap.Error(err))
		return model.Product{}, err
	}

	log.Info("success update product",
		zap.Strings("updated_fields", updatedFields),
		zap.Duration("duration", time.Since(start)),
	)

	return product, nil
}

func (r *repository) BulkCreateVariants(
	ctx context.Context,
	input []*model.NewVariant,
	sellerID string,
) ([]*model.Variant, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "BulkCreateVariants"),
		zap.String("seller_id", sellerID),
		zap.Int("variant_count", len(input)),
	)

	start := time.Now()
	log.Info("start bulk create variants")

	if len(input) > 100 {
		log.Warn("bulk create variants exceeds limit",
			zap.Int("limit", 100),
			zap.Int("received", len(input)),
		)
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

	args := make([]any, 0, len(input)*7)
	valueStrings := make([]string, 0, len(input))

	for i, v := range input {
		idx := i * 7

		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				idx+1, idx+2, idx+3,
				idx+4, idx+5, idx+6, idx+7,
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
		log.Error("failed to execute bulk insert variants", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	variants := make([]*model.Variant, 0, len(input))

	for rows.Next() {
		var v model.Variant
		if err := rows.Scan(
			&v.ID,
			&v.ProductID,
			&v.Name,
			&v.QuantityType,
			&v.Price,
			&v.Stock,
			&v.ImageURL,
			&v.CreatedAt,
		); err != nil {
			log.Error("failed to scan created variant", zap.Error(err))
			return nil, err
		}

		variants = append(variants, &v)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Info("success bulk create variants",
		zap.Int("created_count", len(variants)),
		zap.Duration("duration", time.Since(start)),
	)

	return variants, nil
}

func (r *repository) BulkUpdateVariants(
	ctx context.Context,
	input []*model.UpdateVariant,
	sellerID string,
) ([]*model.Variant, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "BulkUpdateVariants2"),
		zap.String("seller_id", sellerID),
		zap.Int("variant_count", len(input)),
	)

	start := time.Now()
	log.Info("start bulk update variants")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return nil, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

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

		// ✅ Safety guard
		if len(setClauses) == 0 {
			log.Warn("skip update variant: no fields to update",
				zap.String("variant_id", v.ID),
			)
			continue
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
		if err := tx.QueryRowContext(ctx, query, args...).Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.Name,
			&variant.Price,
			&variant.Stock,
			&variant.ImageURL,
			&variant.Description,
		); err != nil {

			log.Error("failed to update variant",
				zap.String("variant_id", v.ID),
				zap.String("product_id", v.ProductID),
				zap.Error(err),
			)
			return nil, err
		}

		updatedVariants = append(updatedVariants, &variant)
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return nil, err
	}

	log.Info("success bulk update variants",
		zap.Int("updated_count", len(updatedVariants)),
		zap.Duration("duration", time.Since(start)),
	)

	return updatedVariants, nil
}

func (r *repository) GetPackages(
	ctx context.Context,
	filter *model.PackageFilterInput,
	sort *model.PackageSortInput,
	limit, page int32,
	includeDisabled bool,
) ([]*model.Package, error) {

	// ---------- PAGINATION ----------
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetPackages"),
		zap.Int32("limit", limit),
		zap.Int32("page", page),
		zap.Int32("offset", offset),
		zap.Bool("include_disabled", includeDisabled),
	)

	log.Debug("start get packages")

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

	args := []any{}
	argIndex := 1

	// ---------- ENABLE / DISABLE ----------
	if !includeDisabled {
		query += fmt.Sprintf(" AND p.status = $%d", argIndex)
		args = append(args, "active")
		argIndex++
	}

	// ---------- FILTERING ----------
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

	// ---------- SORTING ----------
	orderBy := "p.created_at DESC"
	if sort != nil {
		dir := strings.ToUpper(string(sort.Direction))
		if dir != "ASC" && dir != "DESC" {
			dir = "DESC"
		}

		switch sort.Field {
		case model.PackageSortFieldName:
			orderBy = "p.name " + dir
		case model.PackageSortFieldCreatedAt:
			orderBy = "p.created_at " + dir
		}
	}

	query += " ORDER BY " + orderBy

	// ---------- PAGINATION ----------
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	log.Debug("executing query",
		zap.String("query", query),
		zap.Any("args", args),
	)

	// ---------- EXECUTE ----------
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to query packages", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	packagesMap := make(map[string]*model.Package)

	for rows.Next() {
		var (
			p            model.Package
			pi           model.PackageItem
			variantPrice sql.NullFloat64
			imageURL     sql.NullString
			userID       sql.NullString
		)

		if err := rows.Scan(
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
		); err != nil {
			log.Error("failed to scan package row", zap.Error(err))
			return nil, err
		}

		if imageURL.Valid {
			p.ImageURL = &imageURL.String
		}
		if userID.Valid {
			p.UserID = &userID.String
		}

		pkg, exists := packagesMap[p.ID]
		if !exists {
			p.Items = []*model.PackageItem{}
			packagesMap[p.ID] = &p
			pkg = &p
		}

		if pi.ID != "" {
			if variantPrice.Valid {
				pi.Price = variantPrice.Float64
			}
			pkg.Items = append(pkg.Items, &pi)
		}
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	result := make([]*model.Package, 0, len(packagesMap))
	for _, pkg := range packagesMap {
		result = append(result, pkg)
	}

	log.Info("success get packages",
		zap.Int("package_count", len(result)),
	)

	return result, nil
}

func (r *repository) GetProductByID(
	ctx context.Context,
	productParams GetProductOptions,
) (*Product, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetProductByID"),
		zap.String("product_id", productParams.ProductID),
		zap.Bool("only_active", productParams.OnlyActive),
	)

	log.Debug("start get product by id")

	query := `SELECT
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
		COALESCE(sel.name, 'UNKNOWN') as seller_name,
 
		COALESCE(
			json_agg(
				json_build_object(
					'id', v.id,
					'productId', v.product_id,
					'name', v.name,
					'price', v.price,
					'stock', v.stock,
					'imageUrl', v.imageurl,
					'description', v.description
				)
				ORDER BY v.created_at NULLS LAST
			) FILTER (WHERE v.id IS NOT NULL),
			'[]'::json
		) AS variants
	FROM products p
	LEFT JOIN category c ON c.id = p.category_id
	LEFT JOIN subcategories s ON s.id = p.subcategory_id
	LEFT JOIN variants v ON v.product_id = p.id
	LEFT JOIN sellers sel on sel.id = p.seller_id
	WHERE p.id = $1
	`

	var (
		product      Product
		variantsJSON []byte
	)

	args := []any{productParams.ProductID}

	if productParams.OnlyActive {
		query += " AND p.status = $2"
		args = append(args, utils.ProductStatusActive)
	}

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
		s.name,
		sel.name
 	`

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&product.ID,
		&product.Name,
		&product.SellerID,
		&product.CategoryID,
		&product.SubcategoryID,
		&product.Slug,
		&product.ImageURL,
		&product.Description,
		&product.CreatedAt,
		&product.CategoryName,
		&product.SubcategoryName,
		&product.SellerName,
		&variantsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Warn("product not found")
			return nil, ErrProductNotFound // GraphQL-friendly
		}

		log.Error("failed to query product",
			zap.Error(err),
		)
		return nil, ErrRepositoryFailure
	}

	if err := json.Unmarshal(variantsJSON, &product.Variants); err != nil {
		log.Error("failed to unmarshal variants",
			zap.Error(err),
		)
		return nil, ErrRepositoryFailure
	}

	log.Debug("success get product by id",
		zap.Int("variant_count", len(product.Variants)),
	)

	return &product, nil
}

func (r *repository) GetProductVariantByID(
	ctx context.Context,
	opts GetVariantOptions,
) (*Variant, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetProductVariantByID"),
		zap.String("variant_id", opts.VariantID),
		zap.Bool("only_active", opts.OnlyActive),
	)

	log.Debug("start get variant by id")

	query := `
	SELECT
		v.id,
		v.name,
		v.product_id,
		v.quantity_type,
		v.price,
		v.stock,
		v.imageurl,
		p.category_id,
		p.seller_id,
		v.created_at,
		v.description
	FROM variants v
	JOIN products p ON v.product_id = p.id
	WHERE v.id = $1
	`

	args := []any{opts.VariantID}

	if opts.OnlyActive {
		query += " AND p.status = $2"
		args = append(args, utils.ProductStatusActive)
	}

	var variant Variant

	row := r.db.QueryRowContext(ctx, query, args...)
	err := row.Scan(
		&variant.ID,
		&variant.Name,
		&variant.ProductID,
		&variant.QuantityType,
		&variant.Price,
		&variant.Stock,
		&variant.ImageURL,
		&variant.CategoryID,
		&variant.SellerID,
		&variant.CreatedAt,
		&variant.Description,
	)

	if err == sql.ErrNoRows {
		log.Warn("variant not found")
		return nil, nil
	}
	if err != nil {
		log.Error("failed to query variant", zap.Error(err))
		return nil, err
	}

	log.Info("success get variant by id")

	return &variant, nil
}
