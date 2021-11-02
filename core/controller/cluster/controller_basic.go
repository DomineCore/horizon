package cluster

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"g.hz.netease.com/horizon/core/common"
	"g.hz.netease.com/horizon/core/middleware/user"
	"g.hz.netease.com/horizon/lib/q"
	"g.hz.netease.com/horizon/pkg/cluster/cd"
	"g.hz.netease.com/horizon/pkg/cluster/gitrepo"
	"g.hz.netease.com/horizon/pkg/util/errors"
	"g.hz.netease.com/horizon/pkg/util/jsonschema"
	"g.hz.netease.com/horizon/pkg/util/wlog"
)

func (c *controller) ListCluster(ctx context.Context, applicationID uint, environment,
	filter string, query *q.Query) (_ int, _ []*ListClusterResponse, err error) {
	const op = "cluster controller: list cluster"
	defer wlog.Start(ctx, op).Stop(func() string { return wlog.ByErr(err) })

	count, clustersWithEnvAndRegion, err := c.clusterMgr.ListByApplicationAndEnv(ctx,
		applicationID, environment, filter, query)
	if err != nil {
		return 0, nil, errors.E(op, err)
	}

	return count, ofClustersWithEnvAndRegion(clustersWithEnvAndRegion), nil
}

func (c *controller) GetCluster(ctx context.Context, clusterID uint) (_ *GetClusterResponse, err error) {
	const op = "cluster controller: get cluster"
	defer wlog.Start(ctx, op).Stop(func() string { return wlog.ByErr(err) })

	// 1. get cluster from db
	cluster, err := c.clusterMgr.GetByID(ctx, clusterID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 2. get application
	application, err := c.applicationMgr.GetByID(ctx, cluster.ApplicationID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 3. get environmentRegion
	er, err := c.envMgr.GetEnvironmentRegionByID(ctx, cluster.EnvironmentRegionID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 4. get files in git repo
	clusterFiles, err := c.clusterGitRepo.GetCluster(ctx, application.Name, cluster.Name, cluster.Template)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 5. get full path
	group, err := c.groupSvc.GetChildByID(ctx, application.GroupID)
	if err != nil {
		return nil, errors.E(op, err)
	}
	fullPath := fmt.Sprintf("%v/%v/%v", group.FullPath, application.Name, cluster.Name)

	return ofClusterModel(application, cluster, er, fullPath,
		clusterFiles.PipelineJSONBlob, clusterFiles.ApplicationJSONBlob), nil
}

func (c *controller) CreateCluster(ctx context.Context, applicationID uint,
	environment, region string, r *CreateClusterRequest) (_ *GetClusterResponse, err error) {
	const op = "cluster controller: create cluster"
	defer wlog.Start(ctx, op).Stop(func() string { return wlog.ByErr(err) })

	currentUser, err := user.FromContext(ctx)
	if err != nil {
		return nil, errors.E(op, http.StatusInternalServerError,
			errors.ErrorCode(common.InternalError), "no user in context")
	}

	// 1. get application
	application, err := c.applicationMgr.GetByID(ctx, applicationID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 2. validate
	if err := c.validateCreate(ctx, application.Name,
		application.Template, application.TemplateRelease, r); err != nil {
		return nil, errors.E(op, http.StatusBadRequest,
			errors.ErrorCode(common.InvalidRequestBody), err)
	}

	// 3. get environmentRegion
	er, err := c.envMgr.GetByEnvironmentAndRegion(ctx, environment, region)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 4. get regionEntity
	regionEntity, err := c.regionMgr.GetRegionEntity(ctx, region)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 5. get templateRelease
	tr, err := c.templateReleaseMgr.GetByTemplateNameAndRelease(ctx, application.Template, application.TemplateRelease)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 6. create cluster, after created, params.Cluster is the newest cluster
	cluster := r.toClusterModel(application, er)
	cluster.CreatedBy = currentUser.GetID()
	cluster.UpdatedBy = currentUser.GetID()

	// 7. create cluster in git repo
	clusterRepo, err := c.clusterGitRepo.CreateCluster(ctx, &gitrepo.CreateClusterParams{
		BaseParams: &gitrepo.BaseParams{
			Cluster:             cluster.Name,
			PipelineJSONBlob:    r.TemplateInput.Pipeline,
			ApplicationJSONBlob: r.TemplateInput.Application,
			TemplateRelease:     tr,
			Application:         application,
		},
		Environment:  environment,
		RegionEntity: regionEntity,
	})
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 8. create cluster in cd system. todo(gjq) create cluster in cd when deploy
	if err := c.cd.CreateCluster(ctx, &cd.CreateClusterParams{
		Environment:   environment,
		Cluster:       cluster.Name,
		GitRepoSSHURL: clusterRepo.GitRepoSSHURL,
		ValueFiles:    clusterRepo.ValueFiles,
		RegionEntity:  regionEntity,
		Namespace:     clusterRepo.Namespace,
	}); err != nil {
		return nil, errors.E(op, err)
	}

	// 9. create cluster in db
	cluster, err = c.clusterMgr.Create(ctx, cluster)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 10. get full path
	group, err := c.groupSvc.GetChildByID(ctx, application.GroupID)
	if err != nil {
		return nil, errors.E(op, err)
	}
	fullPath := fmt.Sprintf("%v/%v/%v", group.FullPath, application.Name, cluster.Name)

	return ofClusterModel(application, cluster, er, fullPath,
		r.TemplateInput.Pipeline, r.TemplateInput.Application), nil
}

func (c *controller) UpdateCluster(ctx context.Context, clusterID uint,
	r *UpdateClusterRequest) (_ *GetClusterResponse, err error) {
	const op = "cluster controller: update cluster"
	defer wlog.Start(ctx, op).Stop(func() string { return wlog.ByErr(err) })

	currentUser, err := user.FromContext(ctx)
	if err != nil {
		return nil, errors.E(op, http.StatusInternalServerError,
			errors.ErrorCode(common.InternalError), "no user in context")
	}

	// 1. get cluster from db
	cluster, err := c.clusterMgr.GetByID(ctx, clusterID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 2. get application that this cluster belongs to
	application, err := c.applicationMgr.GetByID(ctx, cluster.ApplicationID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 2. validate template input
	var templateRelease string
	if r.Template == nil || r.Template.Release == "" {
		templateRelease = cluster.TemplateRelease
	} else {
		templateRelease = r.Template.Release
	}
	if err := c.validateTemplateInput(ctx,
		cluster.Template, templateRelease, r.TemplateInput); err != nil {
		return nil, errors.E(op, http.StatusBadRequest,
			errors.ErrorCode(common.InvalidRequestBody), fmt.Sprintf("request body validate err: %v", err))
	}

	// 4. get templateRelease
	tr, err := c.templateReleaseMgr.GetByTemplateNameAndRelease(ctx, cluster.Template, templateRelease)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 5. update cluster in git repo
	if err := c.clusterGitRepo.UpdateCluster(ctx, &gitrepo.UpdateClusterParams{
		BaseParams: &gitrepo.BaseParams{
			Cluster:             cluster.Name,
			PipelineJSONBlob:    r.TemplateInput.Pipeline,
			ApplicationJSONBlob: r.TemplateInput.Application,
			TemplateRelease:     tr,
			Application:         application,
		},
	}); err != nil {
		return nil, errors.E(op, err)
	}

	// 6. update cluster in db
	clusterModel := r.toClusterModel(cluster, templateRelease)
	clusterModel.UpdatedBy = currentUser.GetID()
	cluster, err = c.clusterMgr.UpdateByID(ctx, clusterID, clusterModel)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 7. get environmentRegion for this cluster
	er, err := c.envMgr.GetEnvironmentRegionByID(ctx, cluster.EnvironmentRegionID)
	if err != nil {
		return nil, errors.E(op, err)
	}

	// 8. get full path
	group, err := c.groupSvc.GetChildByID(ctx, application.GroupID)
	if err != nil {
		return nil, errors.E(op, err)
	}
	fullPath := fmt.Sprintf("%v/%v/%v", group.FullPath, application.Name, cluster.Name)

	return ofClusterModel(application, cluster, er, fullPath,
		r.TemplateInput.Pipeline, r.TemplateInput.Application), nil
}

// validateCreate validate for create cluster
func (c *controller) validateCreate(ctx context.Context, applicationName,
	template, release string, r *CreateClusterRequest) error {
	if err := validateClusterName(applicationName, r.Name); err != nil {
		return err
	}
	if r.Git == nil || r.Git.Branch == "" {
		return fmt.Errorf("git branch cannot be empty")
	}
	if r.TemplateInput == nil {
		return fmt.Errorf("template input cannot be empty")
	}
	if r.TemplateInput.Application == nil {
		return fmt.Errorf("application config for template cannot be empty")
	}
	if r.TemplateInput.Pipeline == nil {
		return fmt.Errorf("pipeline config for template cannot be empty")
	}
	if err := c.validateTemplateInput(ctx, template, release, r.TemplateInput); err != nil {
		return err
	}
	return nil
}

// validateTemplateInput validate templateInput is valid for template schema
func (c *controller) validateTemplateInput(ctx context.Context,
	template, release string, templateInput *TemplateInput) error {
	schema, err := c.templateSchemaGetter.GetTemplateSchema(ctx, template, release)
	if err != nil {
		return err
	}
	if err := jsonschema.Validate(schema.Application.JSONSchema, templateInput.Application); err != nil {
		return err
	}
	return jsonschema.Validate(schema.Pipeline.JSONSchema, templateInput.Pipeline)
}

// validateClusterName validate cluster name
// 1. name length must be less than 53
// 2. name must match pattern ^(([a-z][-a-z0-9]*)?[a-z0-9])?$
// 3. name must start with application name
func validateClusterName(applicationName, name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 53 {
		return fmt.Errorf("name must not exceed 53 characters")
	}

	// cannot start with a digit.
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("name cannot start with a digit")
	}

	if !strings.HasPrefix(name, applicationName) {
		return fmt.Errorf("cluster name must start with application name")
	}

	pattern := `^(([a-z][-a-z0-9]*)?[a-z0-9])?$`
	r := regexp.MustCompile(pattern)
	if !r.MatchString(name) {
		return fmt.Errorf("invalid cluster name, regex used for validation is %v", pattern)
	}

	return nil
}
