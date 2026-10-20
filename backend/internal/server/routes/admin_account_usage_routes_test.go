package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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

func TestRegisteredRoutesPreservePelicanWithoutCustomIntelligenceTests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{Ops: admin.NewOpsHandler(nil)}}
	v1 := router.Group("/api/v1")
	RegisterAdminRoutes(v1, handlers, func(c *gin.Context) { c.Next() }, func(c *gin.Context) { c.Next() }, func(c *gin.Context) { c.Next() }, nil, nil)
	RegisterUserRoutes(v1, handlers, func(c *gin.Context) { c.Next() }, func(c *gin.Context) { c.Next() }, nil, nil)
	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
		assert.NotContains(t, route.Path, "/account-market", "retired account earnings APIs must not be registered")
	}
	for _, path := range []string{"/api/v1/account-market/upload", "/api/v1/account-market/withdrawals", "/api/v1/admin/account-market/config"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	}
	for _, route := range []string{
		"GET /api/v1/intelligence-tests",
		"GET /api/v1/admin/intelligence-tests",
		"PUT /api/v1/admin/intelligence-tests",
		"POST /api/v1/admin/intelligence-tests/:group_id/run",
	} {
		assert.NotContains(t, registered, route)
	}
	for _, route := range []string{
		"POST /api/v1/admin/accounts/:id/test",
		"POST /api/v1/admin/accounts/:id/pelican-test",
		"GET /api/v1/admin/pelican-test-results",
		"GET /api/v1/admin/pelican-showcase/settings",
		"PUT /api/v1/admin/pelican-showcase/settings",
		"DELETE /api/v1/admin/pelican-showcase/items/:id",
		"GET /api/v1/admin/pelican-group-tests",
		"POST /api/v1/admin/pelican-group-tests",
		"PUT /api/v1/admin/pelican-group-tests/:id",
		"DELETE /api/v1/admin/pelican-group-tests/:id",
		"POST /api/v1/admin/pelican-group-tests/:id/run",
		"GET /api/v1/admin/pelican-group-test-results",
		"GET /api/v1/admin/pelican-group-test-results/:id",
		"GET /api/v1/pelican-showcase",
		"GET /api/v1/pelican-showcase/items/:id",
	} {
		assert.Contains(t, registered, route)
	}
}
