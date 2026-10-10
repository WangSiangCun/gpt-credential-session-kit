package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const channelRequestTimelinesQuery = `
WITH requested_channels AS (
  SELECT channel_id
  FROM unnest($1::bigint[]) AS requested(channel_id)
), recent AS (
  SELECT
    event.channel_id,
    event.actual_cost,
    event.first_token_ms,
    event.input_tokens,
    event.cache_read_tokens,
    event.is_failed,
    event.request_rank
  FROM requested_channels c
  CROSS JOIN LATERAL (
    WITH usage_candidates AS (
      SELECT
        u.channel_id,
        u.actual_cost,
        u.first_token_ms,
        u.input_tokens,
        u.cache_read_tokens,
        FALSE AS is_failed,
        u.created_at,
        u.id AS source_id,
        COALESCE(NULLIF(u.request_id, ''), 'usage:' || u.id::text) AS event_key
      FROM usage_logs u
      -- Express the equality as a single-value range so PostgreSQL reliably
      -- chooses idx_usage_logs_channel_created_id for cold channels too.
      WHERE u.channel_id >= c.channel_id
        AND u.channel_id <= c.channel_id
      ORDER BY u.channel_id, u.created_at DESC, u.id DESC
      LIMIT $2
    ), failure_candidates AS (
      SELECT
        c.channel_id,
        0::numeric AS actual_cost,
        NULL::bigint AS first_token_ms,
        0::bigint AS input_tokens,
        0::bigint AS cache_read_tokens,
        TRUE AS is_failed,
        e.created_at,
        e.id AS source_id,
        COALESCE(
          'client:' || NULLIF(e.client_request_id, ''),
          'local:' || NULLIF(e.request_id, ''),
          'error:' || e.id::text
        ) AS event_key
      FROM channel_groups cg
      CROSS JOIN LATERAL (
        SELECT error_log.*
        FROM ops_error_logs error_log
        WHERE error_log.group_id >= cg.group_id
          AND error_log.group_id <= cg.group_id
          -- status_code is the final response observed by the user. An
          -- upstream failure recovered by failover stays green.
          AND COALESCE(error_log.status_code, 0) >= 400
        ORDER BY error_log.group_id, error_log.created_at DESC, error_log.id DESC
        LIMIT $2
      ) e
      WHERE cg.channel_id = c.channel_id
      ORDER BY e.created_at DESC, e.id DESC
      LIMIT $2
    ), deduplicated AS (
      SELECT DISTINCT ON (candidate.event_key)
        candidate.channel_id,
        candidate.actual_cost,
        candidate.first_token_ms,
        candidate.input_tokens,
        candidate.cache_read_tokens,
        candidate.is_failed,
        candidate.created_at,
        candidate.source_id
      FROM (
        SELECT * FROM usage_candidates
        UNION ALL
        SELECT * FROM failure_candidates
      ) candidate
      ORDER BY
        candidate.event_key,
        candidate.is_failed DESC,
        candidate.created_at DESC,
        candidate.source_id DESC
    )
    SELECT
      deduplicated.channel_id,
      deduplicated.actual_cost,
      deduplicated.first_token_ms,
      deduplicated.input_tokens,
      deduplicated.cache_read_tokens,
      deduplicated.is_failed,
      ROW_NUMBER() OVER (
        ORDER BY deduplicated.created_at DESC, deduplicated.source_id DESC
      ) AS request_rank
    FROM deduplicated
    ORDER BY deduplicated.created_at DESC, deduplicated.source_id DESC
    LIMIT $2
  ) event
), overall AS (
  SELECT
    channel_id,
    COUNT(*)::int AS total_requests,
    AVG(first_token_ms::double precision) FILTER (
      WHERE NOT is_failed AND first_token_ms IS NOT NULL
    ) AS average_first_token_ms,
    COALESCE(AVG(actual_cost::double precision), 0) AS average_actual_cost,
    CASE
      WHEN SUM(input_tokens + cache_read_tokens) > 0
      THEN SUM(cache_read_tokens)::double precision / SUM(input_tokens + cache_read_tokens)::double precision
      ELSE NULL
    END AS average_cache_hit_rate
  FROM recent
  GROUP BY channel_id
)
SELECT
  o.channel_id,
  o.total_requests,
  o.average_first_token_ms,
  o.average_actual_cost,
  o.average_cache_hit_rate,
  (COUNT(*) OVER (PARTITION BY r.channel_id) - r.request_rank)::int AS segment_index,
  1 AS request_count,
  CASE WHEN NOT r.is_failed THEN 1 ELSE 0 END AS success_count,
  CASE WHEN r.is_failed THEN 1 ELSE 0 END AS failed_count,
  CASE WHEN NOT r.is_failed THEN r.first_token_ms::double precision ELSE NULL END AS average_first_token_ms
FROM overall o
JOIN recent r ON r.channel_id = o.channel_id
ORDER BY o.channel_id, segment_index`

// GetChannelRequestTimelines returns one chronological segment per request for
// the latest requestCount requests of every requested channel. Successful
// requests come from usage_logs while client-visible failures come from
// ops_error_logs; selecting from only usage_logs would omit most failures.
//
// The per-channel lateral query limits both sources before merging them, so the
// amount of history sorted for each channel remains bounded by requestCount.
func (r *usageLogRepository) GetChannelRequestTimelines(
	ctx context.Context,
	channelIDs []int64,
	requestCount int,
) (map[int64]*service.ChannelRequestTimeline, error) {
	result := make(map[int64]*service.ChannelRequestTimeline, len(channelIDs))
	if len(channelIDs) == 0 {
		return result, nil
	}
	if requestCount < 1 || requestCount > 5000 {
		return nil, fmt.Errorf("invalid channel timeline request count: %d", requestCount)
	}

	rows, err := r.sql.QueryContext(ctx, channelRequestTimelinesQuery, pq.Array(channelIDs), requestCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			channelID       int64
			totalRequests   int
			avgFirstToken   sql.NullFloat64
			avgActualCost   float64
			avgCacheHitRate sql.NullFloat64
			segmentIndex    int
			requestCount    int
			successCount    int
			failedCount     int
			segmentAvgFirst sql.NullFloat64
		)
		if err := rows.Scan(
			&channelID,
			&totalRequests,
			&avgFirstToken,
			&avgActualCost,
			&avgCacheHitRate,
			&segmentIndex,
			&requestCount,
			&successCount,
			&failedCount,
			&segmentAvgFirst,
		); err != nil {
			return nil, err
		}

		timeline := result[channelID]
		if timeline == nil {
			timeline = &service.ChannelRequestTimeline{
				ChannelID:         channelID,
				TotalRequests:     totalRequests,
				AverageActualCost: avgActualCost,
			}
			if avgFirstToken.Valid {
				value := avgFirstToken.Float64
				timeline.AverageFirstTokenMs = &value
			}
			if avgCacheHitRate.Valid {
				value := avgCacheHitRate.Float64
				timeline.AverageCacheHitRate = &value
			}
			result[channelID] = timeline
		}

		segment := service.ChannelRequestTimelineSegment{
			Index:        segmentIndex,
			RequestCount: requestCount,
			SuccessCount: successCount,
			FailedCount:  failedCount,
		}
		if segmentAvgFirst.Valid {
			value := segmentAvgFirst.Float64
			segment.AverageFirstTokenMs = &value
		}
		timeline.Segments = append(timeline.Segments, segment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
