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

const usLocationID = "b1cf6f36-d768-4505-9f6b-52c872e32b96"

const sdsFilter = `(LOWER(j.title) LIKE '%sds%' OR LOWER(j.title) LIKE '%self-driving%' OR LOWER(j.title) LIKE '%self driving%')
	AND j.location_id = '` + usLocationID + `'
	AND j.title NOT LIKE '[TEST]%'`

const referralFilter = `(
	LOWER(a.source->>'title') LIKE '%referral%'
	OR LOWER(a.source->'sourceType'->>'title') LIKE '%referral%'
)`

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
	where := `WHERE ` + sdsFilter + ` AND ` + referralFilter

	if stage != "" {
		where += fmt.Sprintf(` AND LOWER(CASE
			WHEN a.status = 'Archived' AND a.archive_reason->>'text' IS NOT NULL THEN 'Rejected'
			WHEN a.status = 'Archived' THEN 'Archived'
			ELSE COALESCE(a.current_stage->>'title', 'Submitted')
		END) = LOWER('%s')`, stage)
	}
	if role != "" {
		where += fmt.Sprintf(` AND LOWER(j.title) LIKE LOWER('%%%s%%')`, role)
	}

	return dlClient.query(ctx, `
		SELECT
			a.id AS application_id,
			c.name AS candidate_name,
			c.primary_email,
			j.title AS role,
			j.id AS job_id,
			COALESCE(u.first_name || ' ' || u.last_name, '') AS referrer_name,
			CASE
				WHEN a.status = 'Archived' AND a.archive_reason->>'text' IS NOT NULL THEN 'Rejected'
				WHEN a.status = 'Archived' THEN 'Archived'
				ELSE COALESCE(a.current_stage->>'title', 'Submitted')
			END AS stage,
			a.status AS app_status,
			a.created_at AS applied_at,
			COALESCE(
				o.latest_version->>'startDate',
				o.created_at::text
			) AS hired_at,
			c.company,
			c.title AS current_title,
			a.source->>'title' AS source_name
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		LEFT JOIN ashby_users u ON a.credited_to_user_id = u.id AND a.credited_to_user_id != ''
		LEFT JOIN ashby_offers o ON o.application_id = a.id AND o.acceptance_status = 'Accepted'
		`+where+`
		ORDER BY a.created_at DESC
	`, 60)
}

func dlReferralStats(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			COUNT(*) AS total_referrals,
			COUNT(CASE WHEN a.status != 'Archived' THEN 1 END) AS active,
			COUNT(CASE WHEN a.status = 'Archived' AND a.archive_reason->>'text' IS NOT NULL THEN 1 END) AS rejected
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+` AND `+referralFilter+`
	`, 60)
}

func dlReferralsByStage(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			CASE
				WHEN a.status = 'Archived' AND a.archive_reason->>'text' IS NOT NULL THEN 'Rejected'
				WHEN a.status = 'Archived' THEN 'Archived'
				ELSE COALESCE(a.current_stage->>'title', 'Submitted')
			END AS stage,
			COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+` AND `+referralFilter+`
		GROUP BY stage
		ORDER BY count DESC
	`, 60)
}

func dlReferralsByRole(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT j.title AS role, COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+` AND `+referralFilter+`
		GROUP BY j.title
		ORDER BY count DESC
	`, 60)
}

func dlQuarterlyTrends(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			to_char(date_trunc('quarter', a.created_at), 'YYYY-"Q"Q') AS quarter,
			COUNT(*) AS count
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+` AND `+referralFilter+`
		AND a.created_at >= '2025-01-01'
		GROUP BY date_trunc('quarter', a.created_at)
		ORDER BY quarter
	`, 60)
}

func dlHiredQuarterly(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			to_char(date_trunc('quarter', COALESCE(
				(o.latest_version->>'startDate')::date,
				o.created_at::date,
				a.created_at::date
			)), 'YYYY-"Q"Q') AS quarter,
			COUNT(*) AS hired
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		LEFT JOIN ashby_offers o ON o.application_id = a.id AND o.acceptance_status = 'Accepted'
		WHERE `+sdsFilter+` AND `+referralFilter+`
		AND a.status = 'Hired'
		AND COALESCE(
			(o.latest_version->>'startDate')::date,
			o.created_at::date,
			a.created_at::date
		) >= '2025-01-01'
		GROUP BY date_trunc('quarter', COALESCE(
			(o.latest_version->>'startDate')::date,
			o.created_at::date,
			a.created_at::date
		))
		ORDER BY quarter
	`, 60)
}

func dlHiredReferralList(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			c.name AS candidate_name,
			c.primary_email,
			j.title AS role,
			COALESCE(
				o.latest_version->>'startDate',
				o.created_at::text
			) AS hire_date,
			EXTRACT(YEAR FROM COALESCE(
				(o.latest_version->>'startDate')::date,
				o.created_at::date,
				a.created_at::date
			))::int AS year
		FROM ashby_applications a
		JOIN ashby_candidates c ON a.candidate_id = c.id
		JOIN ashby_jobs j ON a.job_id = j.id
		LEFT JOIN ashby_offers o ON o.application_id = a.id AND o.acceptance_status = 'Accepted'
		WHERE `+sdsFilter+` AND `+referralFilter+`
		AND a.status = 'Hired'
		ORDER BY COALESCE(
			(o.latest_version->>'startDate')::date,
			o.created_at::date,
			a.created_at::date
		) DESC
	`, 60)
}

func dlHiredByRole(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			j.title AS role,
			COUNT(*) AS total_hires,
			COUNT(CASE WHEN `+referralFilter+` THEN 1 END) AS referral_hires
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		WHERE `+sdsFilter+` AND a.status = 'Hired'
		GROUP BY j.title
		ORDER BY total_hires DESC
	`, 60)
}

func dlCompanyReferralComparison(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			d.name AS department,
			to_char(date_trunc('quarter', a.created_at), 'YYYY-"Q"Q') AS quarter,
			COUNT(*) AS referrals
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		JOIN ashby_departments d ON j.department_id = d.id
		WHERE `+referralFilter+`
		AND a.created_at >= '2025-01-01'
		AND j.title NOT LIKE '[TEST]%'
		GROUP BY d.name, date_trunc('quarter', a.created_at)
		ORDER BY department, quarter
	`, 60)
}

func dlReferrerLeaderboard(ctx context.Context) (*dlResponse, error) {
	return dlClient.query(ctx, `
		SELECT
			COALESCE(u.first_name || ' ' || u.last_name, 'Unknown') AS referrer_name,
			COUNT(*) AS referral_count
		FROM ashby_applications a
		JOIN ashby_jobs j ON a.job_id = j.id
		JOIN ashby_users u ON a.credited_to_user_id = u.id
		WHERE `+sdsFilter+` AND `+referralFilter+`
		AND a.created_at >= '2025-01-01'
		AND a.credited_to_user_id IS NOT NULL AND a.credited_to_user_id != ''
		AND u.first_name IS NOT NULL
		GROUP BY referrer_name
		ORDER BY referral_count DESC
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
			j.job_posting_ids,
			COALESCE(ref.cnt, 0) AS referral_count
		FROM ashby_jobs j
		LEFT JOIN (
			SELECT a.job_id, COUNT(*) AS cnt
			FROM ashby_applications a
			WHERE `+referralFilter+`
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
