package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
	"google.golang.org/api/idtoken"
)

const (
	datalakeAPIURL     = "https://datalake.api.experimental.apps.applied.dev"
	datalakeServiceURL = "https://datalake.experimental.apps.applied.dev"
)

const sdsFilter = `(LOWER(j.title) LIKE '%sds%' OR LOWER(j.title) LIKE '%self-driving%' OR LOWER(j.title) LIKE '%self driving%')`

type datalakeClient struct {
	apiKey     string
	httpClient *http.Client
}

var dlClient *datalakeClient

func initDatalakeClient(ctx context.Context) {
	apiKey := os.Getenv("DATALAKE_API_KEY")

	if apiKey != "" {
		dlClient = &datalakeClient{
			apiKey:     apiKey,
			httpClient: &http.Client{},
		}
		zap.L().Info("datalake client initialized (API key)")
		return
	}

	if os.Getenv("K_SERVICE") != "" {
		client, err := idtoken.NewClient(ctx, datalakeServiceURL)
		if err != nil {
			zap.L().Warn("failed to create datalake ID token client", zap.Error(err))
			return
		}
		dlClient = &datalakeClient{httpClient: client}
		zap.L().Info("datalake client initialized (service-to-service)")
		return
	}

	zap.L().Warn("DATALAKE_API_KEY not set, datalake disabled")
}

type dlRequest struct {
	SQL      string `json:"sql"`
	TimeoutS int    `json:"timeout_s,omitempty"`
}

type dlResponse struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"row_count"`
}

func (c *datalakeClient) query(ctx context.Context, sql string, timeoutS int) (*dlResponse, error) {
	body, err := json.Marshal(dlRequest{SQL: sql, TimeoutS: timeoutS})
	if err != nil {
		return nil, err
	}

	baseURL := datalakeServiceURL
	if c.apiKey != "" {
		baseURL = datalakeAPIURL
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("datalake %d: %s", resp.StatusCode, string(respBody))
	}

	var result dlResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Query functions called directly by API handlers ---

func dlListJobs(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			j.id,
			j.title,
			j.status,
			j.department_id,
			j.location_id,
			j.opened_at
		FROM ashby_jobs j
		WHERE j.status = 'Open' AND `+sdsFilter+`
		ORDER BY j.title
	`, 30)
}

func dlListReferrals(ctx context.Context, stage, role string) (*dlResponse, error) {
	where := `WHERE ` + sdsFilter + `
		AND (
			LOWER(c.source_details->>'sourceType') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
		)`

	if stage != "" {
		where += fmt.Sprintf(` AND LOWER(COALESCE(a.current_stage->>'name', 'submitted')) = LOWER('%s')`, stage)
	}
	if role != "" {
		where += fmt.Sprintf(` AND LOWER(j.title) LIKE LOWER('%%%s%%')`, role)
	}

	return dlClient.query(ctx, `
		SELECT
			a.id AS application_id,
			c.name AS candidate_name,
			c.linkedin_url,
			j.title AS role,
			j.id AS job_id,
			COALESCE(u.first_name || ' ' || u.last_name, '') AS referrer_name,
			CASE
				WHEN a.status = 'Archived' AND a.archive_reason->>'reasonTitle' IS NOT NULL THEN 'rejected'
				WHEN a.status = 'Archived' THEN 'archived'
				ELSE COALESCE(a.current_stage->>'name', 'submitted')
			END AS stage,
			a.status AS app_status,
			a.created_at AS applied_at,
			c.company,
			c.title AS current_title
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		LEFT JOIN ashby_users u ON a.credited_to_user_id = u.id
		`+where+`
		ORDER BY a.created_at DESC
	`, 60)
}

func dlReferralStats(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			COUNT(*) AS total_referrals,
			COUNT(CASE WHEN a.status = 'Active' THEN 1 END) AS active,
			COUNT(CASE WHEN a.status = 'Archived' AND a.archive_reason->>'reasonTitle' IS NOT NULL THEN 1 END) AS rejected,
			COUNT(CASE WHEN a.status = 'Archived' AND a.archive_reason->>'reasonTitle' IS NULL THEN 1 END) AS archived
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+`
		AND (
			LOWER(c.source_details->>'sourceType') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
		)
	`, 60)
}

func dlReferralsByStage(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			CASE
				WHEN a.status = 'Archived' AND a.archive_reason->>'reasonTitle' IS NOT NULL THEN 'rejected'
				WHEN a.status = 'Archived' THEN 'archived'
				ELSE COALESCE(a.current_stage->>'name', 'submitted')
			END AS stage,
			COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+`
		AND (
			LOWER(c.source_details->>'sourceType') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
		)
		GROUP BY stage
		ORDER BY count DESC
	`, 60)
}

func dlReferralsByRole(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT j.title AS role, COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+`
		AND (
			LOWER(c.source_details->>'sourceType') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
		)
		GROUP BY j.title
		ORDER BY count DESC
	`, 60)
}

func dlWeeklyTrends(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			to_char(date_trunc('week', a.created_at), 'YYYY-MM-DD') AS week,
			COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+`
		AND (
			LOWER(c.source_details->>'sourceType') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
			OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
		)
		AND a.created_at >= NOW() - INTERVAL '12 weeks'
		GROUP BY date_trunc('week', a.created_at)
		ORDER BY week
	`, 60)
}

func dlJobsWithReferralCounts(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			j.id,
			j.title,
			j.status,
			j.department_id,
			j.location_id,
			j.opened_at,
			COALESCE(ref.cnt, 0) AS referral_count
		FROM ashby_jobs j
		LEFT JOIN (
			SELECT a.job_id, COUNT(*) AS cnt
			FROM ashby_applications a
			JOIN ashby_candidates c ON a.candidate_id = c.id
			WHERE (
				LOWER(c.source_details->>'sourceType') LIKE '%referral%'
				OR LOWER(c.source_details->>'sourceSubtype') LIKE '%referral%'
				OR LOWER(c.source_details->>'sourceName') LIKE '%referral%'
			)
			GROUP BY a.job_id
		) ref ON ref.job_id = j.id
		WHERE j.status = 'Open' AND `+sdsFilter+`
		ORDER BY COALESCE(ref.cnt, 0) DESC, j.title
	`, 60)
}

func toStr(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	default:
		return 0
	}
}
