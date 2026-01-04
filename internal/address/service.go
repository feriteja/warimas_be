package address

import (
	"context"
	"errors"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the business logic for carts.
type Service interface {
	List(ctx context.Context) ([]*Address, error)
	Get(ctx context.Context, addressID uuid.UUID) (*Address, error)

	Create(ctx context.Context, input CreateAddressInput) (*Address, error)
	Update(ctx context.Context, input UpdateAddressInput) (*Address, error)
	Delete(ctx context.Context, addressID uuid.UUID) error

	SetDefaultAddress(ctx context.Context, addressID uuid.UUID) error
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new cart service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(
	ctx context.Context,
) ([]*Address, error) {

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "List"),
		zap.Uint("user_id", userID),
	)

	log.Info("listing addresses")

	return s.repo.GetByUserID(ctx, userID)
}

func (s *service) Get(
	ctx context.Context,
	addressID uuid.UUID,
) (*Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "Get"),
		zap.String("address_id", addressID.String()),
	)

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	addr, err := s.repo.GetByID(ctx, addressID)
	if err != nil {
		log.Error("address not found", zap.Error(err))
		return nil, err
	}

	if addr.UserID != userID || !addr.IsActive {
		log.Warn("unauthorized address access")
		return nil, errors.New("address not found")
	}

	return addr, nil
}

func (s *service) Create(
	ctx context.Context,
	input CreateAddressInput,
) (*Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "Create"),
	)

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	addr := &Address{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      input.Name,
		Phone:     input.Phone,
		Address1:  input.AddressLine1,
		Address2:  input.AddressLine2,
		City:      input.City,
		Province:  input.Province,
		Postal:    input.PostalCode,
		Country:   input.Country,
		IsActive:  true,
		IsDefault: input.SetAsDefault,
	}

	if input.SetAsDefault {
		_ = s.repo.ClearDefault(ctx, userID)
	}

	if err := s.repo.Create(ctx, addr); err != nil {
		log.Error("failed to create address", zap.Error(err))
		return nil, err
	}

	log.Info("address created", zap.String("address_id", addr.ID.String()))
	return addr, nil
}

func (s *service) Update(
	ctx context.Context,
	input UpdateAddressInput,
) (*Address, error) {

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "Update"),
		zap.Uint("user_id", userID),
	)

	oldID, err := uuid.Parse(input.AddressID)
	if err != nil {
		return nil, errors.New("invalid address id")
	}

	oldAddr, err := s.repo.GetByID(ctx, oldID)
	if err != nil || oldAddr.UserID != userID {
		return nil, errors.New("address not found")
	}

	// deactivate old address
	_ = s.repo.Deactivate(ctx, oldID)

	newAddr := &Address{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      input.Name,
		Phone:     input.Phone,
		Address1:  input.AddressLine1,
		Address2:  input.AddressLine2,
		City:      input.City,
		Province:  input.Province,
		Postal:    input.PostalCode,
		Country:   input.Country,
		IsActive:  true,
		IsDefault: input.SetAsDefault,
	}

	if input.SetAsDefault {
		_ = s.repo.ClearDefault(ctx, userID)
	}

	if err := s.repo.Create(ctx, newAddr); err != nil {
		log.Error("failed to update address", zap.Error(err))
		return nil, err
	}

	log.Info("address updated",
		zap.String("old_id", oldID.String()),
		zap.String("new_id", newAddr.ID.String()),
	)

	return newAddr, nil
}

func (s *service) Delete(
	ctx context.Context,
	addressID uuid.UUID,
) error {

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return errors.New("unauthenticated")
	}

	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "Delete"),
		zap.String("address_id", addressID.String()),
		zap.Uint("userID", userID),
	)

	addr, err := s.repo.GetByID(ctx, addressID)
	if err != nil || addr.UserID != userID {
		return errors.New("address not found")
	}

	log.Info("address Deleted",
		zap.String("old_id", addressID.String()),
	)

	return s.repo.Deactivate(ctx, addressID)
}

func (s *service) SetDefaultAddress(
	ctx context.Context,
	addressID uuid.UUID,
) error {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return errors.New("unauthenticated")
	}
	log := logger.FromCtx(ctx).With(
		zap.String("service", "Address"),
		zap.String("method", "SetDefaultAddress"),
		zap.String("address_id", addressID.String()),
		zap.Uint("userID", userID),
	)

	log.Info("setting default address")

	if err := s.repo.SetDefault(ctx, userID, addressID); err != nil {
		log.Error("failed to set default address", zap.Error(err))
		return err
	}

	return nil
}
