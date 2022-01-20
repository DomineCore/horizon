package applicationregion

import (
	"fmt"
	"strconv"

	"g.hz.netease.com/horizon/core/controller/applicationregion"
	envdao "g.hz.netease.com/horizon/pkg/environment/dao"
	perrors "g.hz.netease.com/horizon/pkg/errors"
	"g.hz.netease.com/horizon/pkg/server/response"
	"g.hz.netease.com/horizon/pkg/server/rpcerror"

	"github.com/gin-gonic/gin"
)

const (
	// param
	_applicationIDParam = "applicationID"
)

type API struct {
	applicationRegionCtl applicationregion.Controller
}

func NewAPI(applicationRegionCtl applicationregion.Controller) *API {
	return &API{
		applicationRegionCtl: applicationRegionCtl,
	}
}

func (a *API) List(c *gin.Context) {
	applicationIDStr := c.Param(_applicationIDParam)
	applicationID, err := strconv.ParseUint(applicationIDStr, 10, 0)
	if err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg(err.Error()))
		return
	}
	var res applicationregion.ApplicationRegion
	if res, err = a.applicationRegionCtl.List(c, uint(applicationID)); err != nil {
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsg(err.Error()))
		return
	}
	response.SuccessWithData(c, res)
}

func (a *API) Update(c *gin.Context) {
	applicationIDStr := c.Param(_applicationIDParam)
	applicationID, err := strconv.ParseUint(applicationIDStr, 10, 0)
	if err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg(err.Error()))
		return
	}

	var request applicationregion.ApplicationRegion
	if err := c.ShouldBindJSON(&request); err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg(fmt.Sprintf("request body is invalid, err: %v", err)))
		return
	}

	if err := a.applicationRegionCtl.Update(c, uint(applicationID), request); err != nil {
		switch perrors.Cause(err) {
		case envdao.ErrEnvironmentRegionNotFound:
			response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg(err.Error()))
		default:
			response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsg(err.Error()))
		}
		return
	}

	response.Success(c)
}