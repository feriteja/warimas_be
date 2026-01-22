package packages

import (
	"context"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

type Service interface {
	GetPackages(ctx context.Context, filter *PackageFilterInput, sort *PackageSortInput, limit, page int32) ([]*Package, int64, error)
	AddPackage(ctx context.Context, input CreatePackageInput) (*Package, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetPackages(
	ctx context.Context,
	filter *PackageFilterInput,
	sort *PackageSortInput,
	limit, page int32,
) ([]*Package, int64, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetPackages"),
		zap.Int32("limit", limit),
		zap.Int32("page", page),
	)
	log.Debug("start get packages")

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

	// ---------- AUTH ----------
	role := utils.GetUserRoleFromContext(ctx)
	includeDisabled := role == "ADMIN"

	pkgs, total, err := s.repo.GetPackages(
		ctx,
		filter,
		sort,
		limit,
		page, // Pass page, not offset (repo handles offset calculation)
		includeDisabled,
	)
	if err != nil {
		log.Error("failed to get packages", zap.Error(err))
		return nil, 0, err
	}

	log.Info("success get packages", zap.Int("count", len(pkgs)), zap.Int64("total", total))
	return pkgs, total, nil
}

func (s *service) AddPackage(ctx context.Context, input CreatePackageInput) (*Package, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "AddPackage"),
		zap.String("name", input.Name),
		zap.String("type", input.Type),
	)
	log.Info("start add package")

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("unauthenticated")
		return nil, ErrUnauthenticated
	}
	userRole := utils.GetUserRoleFromContext(ctx)
	if userRole != "ADMIN" && input.Type == "promotion" {
		log.Warn("unauthorized: promotion type requires admin")
		return nil, ErrUnauthorized
	}

	pkg, err := s.repo.CreatePackage(ctx, input, userID)
	if err != nil {
		log.Error("failed to create package", zap.Error(err))
		return nil, err
	}

	log.Info("success create package", zap.String("package_id", pkg.ID))
	return pkg, nil
}
