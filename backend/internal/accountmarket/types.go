package accountmarket

import "time"

const (
	MaxUploadBytes = 2 << 20
	MaxBatchSize   = 20
)

type Config struct {
	Enabled                  bool    `json:"enabled"`
	TargetGroupID            *int64  `json:"target_group_id"`
	TargetGroupName          string  `json:"target_group_name,omitempty"`
	CommissionPercent        float64 `json:"commission_percent"`
	MinimumWithdrawal        float64 `json:"minimum_withdrawal"`
	WithdrawalReviewRequired bool    `json:"withdrawal_review_required"`
}

type UploadInput struct {
	Name           string         `json:"name"`
	Notes          *string        `json:"notes"`
	Platform       string         `json:"platform"`
	Type           string         `json:"type"`
	Credentials    map[string]any `json:"credentials"`
	Extra          map[string]any `json:"extra"`
	Concurrency    int            `json:"concurrency"`
	Priority       int            `json:"priority"`
	RateMultiplier *float64       `json:"rate_multiplier"`
	LoadFactor     *int           `json:"load_factor"`
}

type ProxyOption struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Protocol  string     `json:"protocol"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type UploadResult struct {
	Index     int    `json:"index"`
	Success   bool   `json:"success"`
	AccountID int64  `json:"account_id,omitempty"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	GroupID   int64  `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

type AccountView struct {
	MarketID            int64      `json:"market_id"`
	AccountID           int64      `json:"account_id"`
	UploaderUserID      int64      `json:"uploader_user_id,omitempty"`
	UploaderEmail       string     `json:"uploader_email,omitempty"`
	Name                string     `json:"name"`
	Platform            string     `json:"platform"`
	Type                string     `json:"type"`
	Email               string     `json:"email,omitempty"`
	MarketStatus        string     `json:"market_status"`
	AccountStatus       string     `json:"account_status"`
	Schedulable         bool       `json:"schedulable"`
	ErrorMessage        string     `json:"error_message,omitempty"`
	Concurrency         int        `json:"concurrency"`
	RateMultiplier      float64    `json:"rate_multiplier"`
	GroupID             int64      `json:"group_id"`
	GroupName           string     `json:"group_name"`
	GroupRateMultiplier float64    `json:"group_rate_multiplier"`
	ProxyID             *int64     `json:"proxy_id,omitempty"`
	ProxyName           string     `json:"proxy_name,omitempty"`
	ProxyProtocol       string     `json:"proxy_protocol,omitempty"`
	Quota               any        `json:"quota,omitempty"`
	ResetAt             any        `json:"reset_at,omitempty"`
	LastUsedAt          *time.Time `json:"last_used_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	TotalEarned         float64    `json:"total_earned"`
}

type Summary struct {
	Available    float64 `json:"available"`
	Locked       float64 `json:"locked"`
	Withdrawn    float64 `json:"withdrawn"`
	Total        float64 `json:"total"`
	AccountCount int64   `json:"account_count"`
}

type Withdrawal struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	UserEmail         string     `json:"user_email,omitempty"`
	GrossAmount       float64    `json:"gross_amount"`
	CommissionPercent float64    `json:"commission_percent"`
	CommissionAmount  float64    `json:"commission_amount"`
	NetAmount         float64    `json:"net_amount"`
	PayoutMethod      string     `json:"payout_method"`
	PayoutAccount     string     `json:"payout_account"`
	Status            string     `json:"status"`
	ReviewNote        string     `json:"review_note,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type WithdrawalRequest struct {
	PayoutMethod  string `json:"payout_method"`
	PayoutAccount string `json:"payout_account"`
}

type ReviewRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type AccountUpdateRequest struct {
	Concurrency int `json:"concurrency"`
}
