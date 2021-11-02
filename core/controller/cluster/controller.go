package cluster

import (
	"context"

	"g.hz.netease.com/horizon/lib/q"
	appmanager "g.hz.netease.com/horizon/pkg/application/manager"
	"g.hz.netease.com/horizon/pkg/cluster/cd"
	"g.hz.netease.com/horizon/pkg/cluster/code"
	"g.hz.netease.com/horizon/pkg/cluster/gitrepo"
	clustermanager "g.hz.netease.com/horizon/pkg/cluster/manager"
	envmanager "g.hz.netease.com/horizon/pkg/environment/manager"
	groupsvc "g.hz.netease.com/horizon/pkg/group/service"
	regionmanager "g.hz.netease.com/horizon/pkg/region/manager"
	trmanager "g.hz.netease.com/horizon/pkg/templaterelease/manager"
	templateschema "g.hz.netease.com/horizon/pkg/templaterelease/schema"
)

type Controller interface {
	GetCluster(ctx context.Context, clusterID uint) (*GetClusterResponse, error)
	ListCluster(ctx context.Context, applicationID uint, environment,
		filter string, query *q.Query) (int, []*ListClusterResponse, error)
	CreateCluster(ctx context.Context, applicationID uint, environment, region string,
		request *CreateClusterRequest) (*GetClusterResponse, error)
	UpdateCluster(ctx context.Context, clusterID uint,
		request *UpdateClusterRequest) (*GetClusterResponse, error)
	BuildDeploy(ctx context.Context, request *BuildDeployRequest) (*BuildDeployResponse, error)
}

type controller struct {
	clusterMgr           clustermanager.Manager
	clusterGitRepo       gitrepo.ClusterGitRepo
	commitGetter         code.CommitGetter
	cd                   cd.CD
	applicationMgr       appmanager.Manager
	templateReleaseMgr   trmanager.Manager
	templateSchemaGetter templateschema.Getter
	envMgr               envmanager.Manager
	regionMgr            regionmanager.Manager
	groupSvc             groupsvc.Service
}

var _ Controller = (*controller)(nil)

func NewController(clusterGitRepo gitrepo.ClusterGitRepo, commitGetter code.CommitGetter, cd cd.CD,
	templateSchemaGetter templateschema.Getter) Controller {
	return &controller{
		clusterMgr:           clustermanager.Mgr,
		clusterGitRepo:       clusterGitRepo,
		commitGetter:         commitGetter,
		cd:                   cd,
		applicationMgr:       appmanager.Mgr,
		templateReleaseMgr:   trmanager.Mgr,
		templateSchemaGetter: templateSchemaGetter,
		envMgr:               envmanager.Mgr,
		regionMgr:            regionmanager.Mgr,
		groupSvc:             groupsvc.Svc,
	}
}
