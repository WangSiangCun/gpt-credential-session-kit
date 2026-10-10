package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAccountUsageRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminGroup := router.Group("/api/v1/admin")
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		Account:     &admin.AccountHandler{},
		OpenAIOAuth: &admin.OpenAIOAuthHandler{},
	}}

	registerAccountRoutes(adminGroup, handlers, nil)
	registerOpenAIOAuthRoutes(adminGroup, handlers)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	require.Contains(t, registered, http.MethodPost+" /api/v1/admin/accounts/usage/batch")
	require.Contains(t, registered, http.MethodPost+" /api/v1/admin/openai/accounts/:id/quota/refresh")
	require.Contains(t, registered, http.MethodPost+" /api/v1/admin/accounts/batch-delete")

	// An empty selection must reach validation, without invoking any deletion.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-delete", strings.NewReader(`{"account_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "account_ids is required")
}
