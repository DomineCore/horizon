package context

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

const ParamScope = "scope"

func scopeKey() string {
	return "contextScope"
}

func GetScope(ctx context.Context, req *http.Request) string {
	scope, ok := ScopeFromContext(ctx)
	if !ok {
		return req.URL.Query().Get(ParamScope)
	}
	return scope
}

func ScopeFromContext(ctx context.Context) (string, bool) {
	scope, ok := ctx.Value(scopeKey()).(string)
	return scope, ok
}

func SetScope(c *gin.Context, scope string) {
	// attach cluster scope to context
	c.Set(scopeKey(), scope)
}
