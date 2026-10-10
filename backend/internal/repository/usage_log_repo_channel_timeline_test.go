//go:build unit

package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestGetChannelRequestTimelinesMergesFinalFailures(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	queryShape := `(?s)` +
		regexp.QuoteMeta("WITH usage_candidates AS") + `.*` +
		regexp.QuoteMeta("failure_candidates AS") + `.*` +
		regexp.QuoteMeta("COALESCE(error_log.status_code, 0) >= 400") + `.*` +
		regexp.QuoteMeta("DISTINCT ON (candidate.event_key)") + `.*` +
		regexp.QuoteMeta("UNION ALL")
	mock.ExpectQuery(queryShape).
		WithArgs(pq.Array([]int64{7}), 400).
		WillReturnRows(sqlmock.NewRows([]string{
			"channel_id", "total_requests", "average_first_token_ms", "average_actual_cost",
			"average_cache_hit_rate",
			"segment_index", "request_count", "success_count", "failed_count", "average_first_token_ms",
		}).
			AddRow(int64(7), 2, 120.0, 0.04, 0.25, 0, 1, 1, 0, 120.0).
			AddRow(int64(7), 2, 120.0, 0.04, 0.25, 1, 1, 0, 1, nil))

	timelines, err := repo.GetChannelRequestTimelines(context.Background(), []int64{7}, 400)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Contains(t, timelines, int64(7))
	require.Equal(t, 2, timelines[7].TotalRequests)
	require.NotNil(t, timelines[7].AverageCacheHitRate)
	require.InDelta(t, 0.25, *timelines[7].AverageCacheHitRate, 0.000001)
	require.Len(t, timelines[7].Segments, 2)
	require.Equal(t, 1, timelines[7].Segments[0].SuccessCount)
	require.Equal(t, 1, timelines[7].Segments[1].FailedCount)
}

func TestChannelRequestTimelinesQueryIgnoresRecoveredUpstreamFailures(t *testing.T) {
	require.Contains(t, channelRequestTimelinesQuery, "COALESCE(error_log.status_code, 0) >= 400")
	require.NotContains(t, channelRequestTimelinesQuery, "upstream_status_code")
}

func TestChannelRequestTimelinesQueryUsesUsageLogRequestIDPrefixes(t *testing.T) {
	require.Contains(t, channelRequestTimelinesQuery, "'client:' || NULLIF(e.client_request_id, '')")
	require.Contains(t, channelRequestTimelinesQuery, "'local:' || NULLIF(e.request_id, '')")
}

func TestChannelRequestTimelinesQueryCalculatesTokenWeightedCacheHitRate(t *testing.T) {
	require.Contains(t, channelRequestTimelinesQuery, "SUM(cache_read_tokens)::double precision / SUM(input_tokens + cache_read_tokens)::double precision")
	require.Contains(t, channelRequestTimelinesQuery, "WHEN SUM(input_tokens + cache_read_tokens) > 0")
}

func TestGetChannelRequestTimelinesRejectsInvalidCountWithoutQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	_, err := repo.GetChannelRequestTimelines(context.Background(), []int64{7}, 0)

	require.ErrorContains(t, err, "invalid channel timeline request count")
	require.NoError(t, mock.ExpectationsWereMet())
}
