package packages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"warimas-be/internal/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Repository interface {
	GetPackages(ctx context.Context, filter *PackageFilterInput, sort *PackageSortInput, limit, page int32, includeDisabled bool) ([]*Package, int64, error)
	CreatePackage(ctx context.Context, input CreatePackageInput, userID uint) (*Package, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetPackages(
	ctx context.Context,
	filter *PackageFilterInput,
	sort *PackageSortInput,
	limit, page int32,
	includeDisabled bool,
) ([]*Package, int64, error) {

	// ---------- PAGINATION ----------
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if page <= 0 {
		page = 1
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

	// Base conditions
	whereClause := " WHERE p.deleted_at IS NULL" // Always hide soft-deleted items
	args := []any{}
	argIndex := 1

	// ---------- ENABLE / DISABLE ----------
	if !includeDisabled {
		whereClause += " AND p.is_active = TRUE"
	}

	// ---------- FILTERING ----------
	if filter != nil {
		if filter.ID != nil {
			whereClause += fmt.Sprintf(" AND p.id = $%d", argIndex)
			args = append(args, *filter.ID)
			argIndex++
		}

		if filter.Name != nil && *filter.Name != "" {
			whereClause += fmt.Sprintf(" AND p.name ILIKE $%d", argIndex)
			args = append(args, "%"+*filter.Name+"%")
			argIndex++
		}
	}

	// ---------- COUNT ----------
	var total int64
	countQuery := "SELECT COUNT(DISTINCT p.id) FROM packages p" + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		log.Error("failed to count packages", zap.Error(err))
		return nil, 0, err
	}

	// ---------- SORTING ----------
	orderBy := "p.created_at DESC"
	if sort != nil {
		dir := sort.Direction

		switch sort.Field {
		case PackageSortFieldName:
			orderBy = fmt.Sprintf("p.name %s", dir)
		case PackageSortFieldCreatedAt:
			orderBy = fmt.Sprintf("p.created_at %s", dir)
		}
	}

	// ---------- QUERY ----------
	query := `
		SELECT
			p.id,
			p.name,
			p.image_url,
			p.user_id,
			p.type,
			p.created_at,
			p.updated_at,
			pi.id,
			pi.variant_id,
			pi.name,
			pi.image_url,
			pi.price,
			pi.quantity,
			pi.created_at,
			pi.updated_at
		FROM packages p
		LEFT JOIN package_items pi ON p.id = pi.package_id
	` + whereClause + " ORDER BY " + orderBy + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, limit, offset)

	log.Debug("executing query",
		zap.String("query", query),
		zap.Any("args", args),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to query packages", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	packagesMap := make(map[string]*Package)
	result := []*Package{}

	for rows.Next() {
		var (
			pID, pName string
			pImageURL  sql.NullString
			pUserID    sql.NullInt64
			pType      sql.NullString
			pCreatedAt time.Time
			pUpdatedAt time.Time

			itemID        sql.NullString
			itemVariantID sql.NullString
			itemName      sql.NullString
			itemImageURL  sql.NullString
			itemPrice     sql.NullFloat64
			itemQuantity  sql.NullInt32
			itemCreatedAt sql.NullTime
			itemUpdatedAt sql.NullTime
		)

		if err := rows.Scan(
			&pID,
			&pName,
			&pImageURL,
			&pUserID,
			&pType,
			&pCreatedAt,
			&pUpdatedAt,
			&itemID,
			&itemVariantID,
			&itemName,
			&itemImageURL,
			&itemPrice,
			&itemQuantity,
			&itemCreatedAt,
			&itemUpdatedAt,
		); err != nil {
			log.Error("failed to scan package row", zap.Error(err))
			return nil, 0, err
		}

		pkg, exists := packagesMap[pID]
		if !exists {
			var uid *uint
			if pUserID.Valid {
				u := uint(pUserID.Int64)
				uid = &u
			}
			var img *string
			if pImageURL.Valid {
				s := pImageURL.String
				img = &s
			}

			pkg = &Package{
				ID:        pID,
				Name:      pName,
				Type:      pType.String,
				ImageURL:  img,
				UserID:    uid,
				Items:     []*PackageItem{},
				CreatedAt: pCreatedAt.Format(time.RFC3339),
				UpdatedAt: pUpdatedAt.Format(time.RFC3339),
			}
			packagesMap[pID] = pkg
			result = append(result, pkg)
		}

		if itemID.Valid {
			item := &PackageItem{
				ID:        itemID.String,
				PackageID: pID,
				VariantID: itemVariantID.String,
				Name:      itemName.String,
				ImageURL:  itemImageURL.String,
				Price:     itemPrice.Float64,
				Quantity:  itemQuantity.Int32,
			}
			if itemCreatedAt.Valid {
				item.CreatedAt = itemCreatedAt.Time.Format(time.RFC3339)
			}
			if itemUpdatedAt.Valid {
				item.UpdatedAt = itemUpdatedAt.Time.Format(time.RFC3339)
			}
			pkg.Items = append(pkg.Items, item)
		}
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, 0, err
	}

	log.Info("success get packages",
		zap.Int("package_count", len(result)),
	)

	return result, total, nil
}

func (r *repository) CreatePackage(ctx context.Context, input CreatePackageInput, userID uint) (*Package, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "CreatePackage"),
	)
	log.Debug("start create package transaction")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	pkgID := uuid.New().String()
	now := time.Now()

	// Insert Package
	_, err = tx.ExecContext(ctx, `
		INSERT INTO packages (id, name, type, user_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pkgID, input.Name, input.Type, userID, true, now, now)
	if err != nil {
		log.Error("failed to insert package", zap.Error(err))
		if strings.Contains(err.Error(), "chk_packages_type") {
			return nil, errors.New("invalid package type")
		}
		return nil, errors.New("failed to create package")
	}

	// Insert Items
	items := make([]*PackageItem, 0, len(input.Items))
	for _, item := range input.Items {
		itemID := uuid.New().String()

		var vName, vImage string
		var vPrice float64
		err := tx.QueryRowContext(ctx, "SELECT name, imageurl, price FROM variants WHERE id = $1", item.VariantID).Scan(&vName, &vImage, &vPrice)
		if err != nil {
			log.Error("failed to get variant for package item", zap.String("variant_id", item.VariantID), zap.Error(err))
			return nil, fmt.Errorf("variant not found: %v", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO package_items (id, package_id, variant_id, name, image_url,  quantity, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, itemID, pkgID, item.VariantID, vName, vImage, item.Quantity, now, now)
		if err != nil {
			log.Error("failed to insert package item", zap.Error(err))
			return nil, err
		}

		items = append(items, &PackageItem{
			ID:        itemID,
			PackageID: pkgID,
			VariantID: item.VariantID,
			Name:      vName,
			ImageURL:  vImage,
			Price:     vPrice,
			Quantity:  item.Quantity,
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		})
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return nil, err
	}

	log.Info("success create package", zap.String("package_id", pkgID), zap.Int("items_count", len(items)))

	return &Package{
		ID:        pkgID,
		Name:      input.Name,
		Type:      input.Type,
		UserID:    &userID,
		Items:     items,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}, nil
}
