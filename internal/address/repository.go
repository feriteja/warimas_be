package address

import (
	"context"
	"database/sql"
	"errors"
	"warimas-be/internal/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Repository interface {
	GetByUserID(ctx context.Context, userID uint) ([]*Address, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Address, error)

	Create(ctx context.Context, addr *Address) error
	Deactivate(ctx context.Context, id uuid.UUID) error

	ClearDefault(ctx context.Context, userID uint) error
	SetDefault(ctx context.Context, userID uint, addressID uuid.UUID) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByUserID(
	ctx context.Context,
	userID uint,
) ([]*Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "GetByUserID"),
		zap.Uint("user_id", userID),
	)

	const q = `
		SELECT
			id, user_id,
			name, phone,
			address_line1, address_line2,
			city, province, postal_code, country,
			is_default, is_active, receiver_name
		FROM addresses
		WHERE user_id = $1
		  AND is_active = true
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		log.Error("query failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var res []*Address

	for rows.Next() {
		var a Address
		if err := rows.Scan(
			&a.ID, &a.UserID,
			&a.Name, &a.Phone,
			&a.Address1, &a.Address2,
			&a.City, &a.Province, &a.Postal, &a.Country,
			&a.IsDefault, &a.IsActive, &a.ReceiverName,
		); err != nil {
			log.Error("scan failed", zap.Error(err))
			return nil, err
		}
		res = append(res, &a)
	}

	return res, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "GetByID"),
		zap.String("address_id", id.String()),
	)

	const q = `
		SELECT
			id, user_id,
			name, phone,
			address_line1, address_line2,
			city, province, postal_code, country,
			is_default, is_active, receiver_name
		FROM addresses
		WHERE id = $1 AND is_active = true
		LIMIT 1
	`

	var a Address
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.UserID,
		&a.Name, &a.Phone,
		&a.Address1, &a.Address2,
		&a.City, &a.Province, &a.Postal, &a.Country,
		&a.IsDefault, &a.IsActive, &a.ReceiverName,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("address not found")
	}
	if err != nil {
		log.Error("query failed", zap.Error(err))
		return nil, err
	}

	return &a, nil
}

func (r *repository) Create(
	ctx context.Context,
	addr *Address,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "Create"),
		zap.String("address_id", addr.ID.String()),
	)

	const q = `
		INSERT INTO addresses (
			id, user_id,
			name, phone,
			address_line1, address_line2,
			city, province, postal_code, country,
			is_default, is_active
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6,
			$7, $8, $9, $10,
			$11, $12
		)
	`

	_, err := r.db.ExecContext(
		ctx, q,
		addr.ID, addr.UserID,
		addr.Name, addr.Phone,
		addr.Address1, addr.Address2,
		addr.City, addr.Province, addr.Postal, addr.Country,
		addr.IsDefault, addr.IsActive,
	)

	if err != nil {
		log.Error("insert failed", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Deactivate(
	ctx context.Context,
	id uuid.UUID,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "Deactivate"),
		zap.String("address_id", id.String()),
	)
	log.Debug("Start deactivating address")

	const q = `
		UPDATE addresses
		SET is_active = false,
		    is_default = false
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) ClearDefault(
	ctx context.Context,
	userID uint,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "ClearDefault"),
		zap.Uint("user_id", userID),
	)

	log.Debug("start clearing default address")

	const q = `
		UPDATE addresses
		SET is_default = false
		WHERE user_id = $1
		  AND is_default = true
	`

	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}

func (r *repository) SetDefault(
	ctx context.Context,
	userID uint,
	addressID uuid.UUID,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("repo", "Address"),
		zap.String("method", "SetDefault"),
		zap.Uint("user_id", userID),
		zap.String("address_id", addressID.String()),
	)

	const q = `
		UPDATE addresses
		SET is_default = true
		WHERE user_id = $1
		  AND id = $2
		  AND is_active = true
	`

	log.Debug("Start setting default address")

	_, err := r.db.ExecContext(ctx, q, userID, addressID)
	return err
}
