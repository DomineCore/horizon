package manager

import (
	"context"

	herrors "g.hz.netease.com/horizon/core/errors"
	"g.hz.netease.com/horizon/pkg/environment/dao"
	"g.hz.netease.com/horizon/pkg/environment/models"
	perror "g.hz.netease.com/horizon/pkg/errors"
	regiondao "g.hz.netease.com/horizon/pkg/region/dao"
	regionmodels "g.hz.netease.com/horizon/pkg/region/models"
)

var (
	// Mgr is the global environment manager
	Mgr = New()
)

func New() Manager {
	return &manager{
		envDAO:    dao.NewDAO(),
		regionDAO: regiondao.NewDAO(),
	}
}

type Manager interface {
	EnvironmentManager
	EnvironmentRegionManager
}

type EnvironmentManager interface {
	// CreateEnvironment create a environment
	CreateEnvironment(ctx context.Context, environment *models.Environment) (*models.Environment, error)
	// ListAllEnvironment list all environments
	ListAllEnvironment(ctx context.Context) ([]*models.Environment, error)
	// UpdateByID update environment by id
	UpdateByID(ctx context.Context, id uint, environment *models.Environment) error
}

type EnvironmentRegionManager interface {
	// CreateEnvironmentRegion create a environmentRegion
	CreateEnvironmentRegion(ctx context.Context, er *models.EnvironmentRegion) (*models.EnvironmentRegion, error)
	// ListRegionsByEnvironment list regions by env
	ListRegionsByEnvironment(ctx context.Context, env string) ([]*regionmodels.Region, error)
	GetEnvironmentRegionByID(ctx context.Context, id uint) (*models.EnvironmentRegion, error)
	GetByEnvironmentAndRegion(ctx context.Context, env, region string) (*models.EnvironmentRegion, error)
}

type manager struct {
	envDAO    dao.DAO
	regionDAO regiondao.DAO
}

func (m *manager) UpdateByID(ctx context.Context, id uint, environment *models.Environment) error {
	return m.envDAO.UpdateByID(ctx, id, environment)
}

func (m *manager) CreateEnvironment(ctx context.Context, environment *models.Environment) (*models.Environment, error) {
	return m.envDAO.CreateEnvironment(ctx, environment)
}

func (m *manager) ListAllEnvironment(ctx context.Context) ([]*models.Environment, error) {
	return m.envDAO.ListAllEnvironment(ctx)
}

func (m *manager) CreateEnvironmentRegion(ctx context.Context,
	er *models.EnvironmentRegion) (*models.EnvironmentRegion, error) {
	_, err := m.envDAO.GetEnvironmentRegionByEnvAndRegion(ctx,
		er.EnvironmentName, er.RegionName)
	if err != nil {
		if e, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); !ok || e.Source != herrors.EnvironmentRegionInDB {
			return nil, err
		}
	} else {
		return nil, perror.Wrap(herrors.ErrNameConflict, "already exists")
	}

	return m.envDAO.CreateEnvironmentRegion(ctx, er)
}

func (m *manager) ListRegionsByEnvironment(ctx context.Context, env string) ([]*regionmodels.Region, error) {
	regionNames, err := m.envDAO.ListRegionsByEnvironment(ctx, env)
	if err != nil {
		return nil, err
	}

	regions, err := m.regionDAO.ListByNames(ctx, regionNames)
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func (m *manager) GetEnvironmentRegionByID(ctx context.Context, id uint) (*models.EnvironmentRegion, error) {
	return m.envDAO.GetEnvironmentRegionByID(ctx, id)
}

func (m *manager) GetByEnvironmentAndRegion(ctx context.Context,
	env, region string) (*models.EnvironmentRegion, error) {
	return m.envDAO.GetEnvironmentRegionByEnvAndRegion(ctx, env, region)
}
