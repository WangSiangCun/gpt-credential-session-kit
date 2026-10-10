package accountmarket

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(ProvideService, NewHandler)

func ProvideService(db *sql.DB, admin service.AdminService, tester *service.AccountTestService, oauth *service.OpenAIOAuthService, quota *service.OpenAIQuotaService, recoverer *service.RateLimitService) *Service {
	return NewService(db, admin, tester, oauth, quota, recoverer)
}
