package router

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDistributorAdminModelDiscountTemplateExportRouteRegistered(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	engine := gin.New()
	SetApiRouter(engine)

	const path = "/api/distributor/admin/model-discount-template/export"
	for _, route := range engine.Routes() {
		if route.Method == http.MethodGet && route.Path == path {
			if !strings.HasSuffix(route.Handler, ".ExportDistributorModelDiscountTemplateAdmin") {
				t.Fatalf("handler = %q", route.Handler)
			}
			return
		}
	}
	t.Fatalf("GET %s is not registered", path)
}
