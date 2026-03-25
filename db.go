package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var db *pgxpool.Pool

// --- Models ---

type Job struct {
	GreenhouseID  int64      `json:"greenhouse_id"`
	Title         string     `json:"title"`
	Department    *string    `json:"department"`
	Team          *string    `json:"team"`
	Status        string     `json:"status"`
	IsPriority    bool       `json:"is_priority"`
	ReferralCount int        `json:"referral_count"`
	Location      *string    `json:"location"`
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Referral struct {
	ID                      int64      `json:"id"`
	CandidateName           string     `json:"candidate_name"`
	LinkedInURL             *string    `json:"linkedin_url"`
	Role                    *string    `json:"role"`
	JobGreenhouseID         *int64     `json:"job_greenhouse_id"`
	ReferrerName            *string    `json:"referrer_name"`
	ReferrerSlackID         *string    `json:"referrer_slack_id"`
	Stage                   string     `json:"stage"`
	GreenhouseCandidateID   *int64     `json:"greenhouse_candidate_id"`
	GreenhouseApplicationID *int64     `json:"greenhouse_application_id"`
	SlackChannelID          *string    `json:"slack_channel_id"`
	SlackMessageTS          *string    `json:"slack_message_ts"`
	Source                  string     `json:"source"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type SyncLogEntry struct {
	ID            int64      `json:"id"`
	SyncType      string     `json:"sync_type"`
	Status        string     `json:"status"`
	RecordsSynced int        `json:"records_synced"`
	ErrorMessage  *string    `json:"error_message"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
}

type StageCount struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

type RoleCount struct {
	Role  string `json:"role"`
	Count int    `json:"count"`
}

type WeeklyCount struct {
	Week  string `json:"week"`
	Count int    `json:"count"`
}

// --- Connection ---

func ConnectDB(ctx context.Context) error {
	dbUser := os.Getenv("DB_USER")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "postgres"
	}

	instanceConn := os.Getenv("INSTANCE_CONNECTION_NAME")
	if instanceConn == "" {
		dsn := fmt.Sprintf("host=localhost port=5432 user=%s dbname=%s sslmode=disable", dbUser, dbName)
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to local database: %w", err)
		}
		db = pool
	} else {
		dialer, err := cloudsqlconn.NewDialer(ctx,
			cloudsqlconn.WithIAMAuthN(),
			cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
		if err != nil {
			return fmt.Errorf("failed to create Cloud SQL dialer: %w", err)
		}

		config, err := pgxpool.ParseConfig(fmt.Sprintf(
			"user=%s dbname=%s sslmode=disable", dbUser, dbName))
		if err != nil {
			return fmt.Errorf("failed to parse database config: %w", err)
		}

		config.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.Dial(ctx, instanceConn)
		}

		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to connect to Cloud SQL: %w", err)
		}
		db = pool
	}

	schema := strings.ReplaceAll(os.Getenv("K_SERVICE"), "-", "_")
	if schema != "" {
		if _, err := db.Exec(ctx, "SET search_path TO "+schema); err != nil {
			zap.L().Warn("failed to set search_path", zap.Error(err))
		}
	}

	if err := runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runMigrations(ctx context.Context) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS jobs (
		greenhouse_id BIGINT PRIMARY KEY,
		title TEXT NOT NULL,
		department TEXT,
		team TEXT,
		status TEXT DEFAULT 'open',
		is_priority BOOLEAN DEFAULT FALSE,
		referral_count INTEGER DEFAULT 0,
		location TEXT,
		last_synced_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS referrals (
		id SERIAL PRIMARY KEY,
		candidate_name TEXT NOT NULL,
		linkedin_url TEXT,
		role TEXT,
		job_greenhouse_id BIGINT REFERENCES jobs(greenhouse_id),
		referrer_name TEXT,
		referrer_slack_id TEXT,
		stage TEXT DEFAULT 'submitted',
		greenhouse_candidate_id BIGINT,
		greenhouse_application_id BIGINT,
		slack_channel_id TEXT,
		slack_message_ts TEXT,
		source TEXT DEFAULT 'slack',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_referrals_stage ON referrals(stage);
	CREATE INDEX IF NOT EXISTS idx_referrals_role ON referrals(role);
	CREATE INDEX IF NOT EXISTS idx_referrals_job ON referrals(job_greenhouse_id);
	CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals(referrer_slack_id);
	CREATE INDEX IF NOT EXISTS idx_referrals_created ON referrals(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_referrals_gh_candidate ON referrals(greenhouse_candidate_id);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(is_priority);
	CREATE INDEX IF NOT EXISTS idx_jobs_department ON jobs(department);

	CREATE TABLE IF NOT EXISTS sync_log (
		id SERIAL PRIMARY KEY,
		sync_type TEXT NOT NULL,
		status TEXT NOT NULL,
		records_synced INTEGER DEFAULT 0,
		error_message TEXT,
		started_at TIMESTAMP DEFAULT NOW(),
		finished_at TIMESTAMP
	);
	`
	_, err := db.Exec(ctx, ddl)
	return err
}

// --- Job CRUD ---

func UpsertJob(ctx context.Context, j *Job) error {
	query := `
		INSERT INTO jobs (greenhouse_id, title, department, team, status, location, referral_count, last_synced_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (greenhouse_id) DO UPDATE SET
			title = EXCLUDED.title,
			department = EXCLUDED.department,
			team = EXCLUDED.team,
			status = EXCLUDED.status,
			location = EXCLUDED.location,
			referral_count = EXCLUDED.referral_count,
			last_synced_at = NOW(),
			updated_at = NOW()
	`
	_, err := db.Exec(ctx, query, j.GreenhouseID, j.Title, j.Department, j.Team, j.Status, j.Location, j.ReferralCount)
	return err
}

func ListJobs(ctx context.Context, statusFilter string, priorityOnly bool) ([]Job, error) {
	query := `SELECT greenhouse_id, title, department, team, status, is_priority, referral_count, location, last_synced_at, created_at, updated_at FROM jobs WHERE 1=1`
	args := []interface{}{}
	n := 1

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, statusFilter)
		n++
	}
	if priorityOnly {
		query += " AND is_priority = TRUE"
	}

	query += " ORDER BY is_priority DESC, referral_count DESC, title ASC"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.GreenhouseID, &j.Title, &j.Department, &j.Team, &j.Status, &j.IsPriority, &j.ReferralCount, &j.Location, &j.LastSyncedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func SetJobPriority(ctx context.Context, ghID int64, priority bool) error {
	_, err := db.Exec(ctx, `UPDATE jobs SET is_priority = $1, updated_at = NOW() WHERE greenhouse_id = $2`, priority, ghID)
	return err
}

func MarkClosedJobsMissing(ctx context.Context, activeIDs []int64) error {
	if len(activeIDs) == 0 {
		_, err := db.Exec(ctx, `UPDATE jobs SET status = 'closed', updated_at = NOW() WHERE status = 'open'`)
		return err
	}
	_, err := db.Exec(ctx, `UPDATE jobs SET status = 'closed', updated_at = NOW() WHERE status = 'open' AND greenhouse_id != ALL($1)`, activeIDs)
	return err
}

// --- Referral CRUD ---

func UpsertReferral(ctx context.Context, r *Referral) error {
	if r.GreenhouseCandidateID != nil {
		query := `
			INSERT INTO referrals (candidate_name, linkedin_url, role, job_greenhouse_id, referrer_name, referrer_slack_id, stage, greenhouse_candidate_id, greenhouse_application_id, slack_channel_id, slack_message_ts, source, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
			ON CONFLICT (id) DO UPDATE SET
				stage = EXCLUDED.stage,
				role = EXCLUDED.role,
				job_greenhouse_id = EXCLUDED.job_greenhouse_id,
				updated_at = NOW()
			RETURNING id
		`
		return db.QueryRow(ctx, query,
			r.CandidateName, r.LinkedInURL, r.Role, r.JobGreenhouseID,
			r.ReferrerName, r.ReferrerSlackID, r.Stage,
			r.GreenhouseCandidateID, r.GreenhouseApplicationID,
			r.SlackChannelID, r.SlackMessageTS, r.Source,
		).Scan(&r.ID)
	}

	query := `
		INSERT INTO referrals (candidate_name, linkedin_url, role, job_greenhouse_id, referrer_name, referrer_slack_id, stage, greenhouse_candidate_id, greenhouse_application_id, slack_channel_id, slack_message_ts, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		RETURNING id
	`
	return db.QueryRow(ctx, query,
		r.CandidateName, r.LinkedInURL, r.Role, r.JobGreenhouseID,
		r.ReferrerName, r.ReferrerSlackID, r.Stage,
		r.GreenhouseCandidateID, r.GreenhouseApplicationID,
		r.SlackChannelID, r.SlackMessageTS, r.Source,
	).Scan(&r.ID)
}

func UpsertReferralByGHCandidate(ctx context.Context, r *Referral) error {
	query := `
		INSERT INTO referrals (candidate_name, linkedin_url, role, job_greenhouse_id, referrer_name, stage, greenhouse_candidate_id, greenhouse_application_id, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (greenhouse_candidate_id) WHERE greenhouse_candidate_id IS NOT NULL DO UPDATE SET
			stage = EXCLUDED.stage,
			role = EXCLUDED.role,
			job_greenhouse_id = EXCLUDED.job_greenhouse_id,
			referrer_name = COALESCE(EXCLUDED.referrer_name, referrals.referrer_name),
			updated_at = NOW()
		RETURNING id
	`
	return db.QueryRow(ctx, query,
		r.CandidateName, r.LinkedInURL, r.Role, r.JobGreenhouseID,
		r.ReferrerName, r.Stage,
		r.GreenhouseCandidateID, r.GreenhouseApplicationID, r.Source,
	).Scan(&r.ID)
}

func ListReferrals(ctx context.Context, stage, role string, jobID *int64) ([]Referral, error) {
	query := `SELECT id, candidate_name, linkedin_url, role, job_greenhouse_id, referrer_name, referrer_slack_id, stage, greenhouse_candidate_id, greenhouse_application_id, slack_channel_id, slack_message_ts, source, created_at, updated_at FROM referrals WHERE 1=1`
	args := []interface{}{}
	n := 1

	if stage != "" {
		query += fmt.Sprintf(" AND stage = $%d", n)
		args = append(args, stage)
		n++
	}
	if role != "" {
		query += fmt.Sprintf(" AND role ILIKE $%d", n)
		args = append(args, "%"+role+"%")
		n++
	}
	if jobID != nil {
		query += fmt.Sprintf(" AND job_greenhouse_id = $%d", n)
		args = append(args, *jobID)
		n++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []Referral
	for rows.Next() {
		r, err := scanReferral(rows)
		if err != nil {
			return nil, err
		}
		referrals = append(referrals, *r)
	}
	return referrals, rows.Err()
}

func scanReferral(rows pgx.Rows) (*Referral, error) {
	var r Referral
	err := rows.Scan(
		&r.ID, &r.CandidateName, &r.LinkedInURL, &r.Role, &r.JobGreenhouseID,
		&r.ReferrerName, &r.ReferrerSlackID, &r.Stage,
		&r.GreenhouseCandidateID, &r.GreenhouseApplicationID,
		&r.SlackChannelID, &r.SlackMessageTS, &r.Source,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Stats queries ---

func GetReferralStats(ctx context.Context) (map[string]interface{}, error) {
	stageRows, err := db.Query(ctx, `SELECT stage, COUNT(*) FROM referrals GROUP BY stage`)
	if err != nil {
		return nil, err
	}
	defer stageRows.Close()

	stages := map[string]int{}
	total := 0
	for stageRows.Next() {
		var s string
		var c int
		if err := stageRows.Scan(&s, &c); err != nil {
			return nil, err
		}
		stages[s] = c
		total += c
	}
	if err := stageRows.Err(); err != nil {
		return nil, err
	}

	var openJobs int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE status = 'open'`).Scan(&openJobs)

	var priorityJobs int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE is_priority = TRUE AND status = 'open'`).Scan(&priorityJobs)

	return map[string]interface{}{
		"total_referrals": total,
		"stages":          stages,
		"open_jobs":       openJobs,
		"priority_jobs":   priorityJobs,
	}, nil
}

func GetReferralsByStage(ctx context.Context) ([]StageCount, error) {
	rows, err := db.Query(ctx, `SELECT stage, COUNT(*) FROM referrals GROUP BY stage ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []StageCount
	for rows.Next() {
		var sc StageCount
		if err := rows.Scan(&sc.Stage, &sc.Count); err != nil {
			return nil, err
		}
		counts = append(counts, sc)
	}
	return counts, rows.Err()
}

func GetReferralsByRole(ctx context.Context) ([]RoleCount, error) {
	rows, err := db.Query(ctx, `SELECT COALESCE(role, 'Unspecified') AS role, COUNT(*) FROM referrals GROUP BY role ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []RoleCount
	for rows.Next() {
		var rc RoleCount
		if err := rows.Scan(&rc.Role, &rc.Count); err != nil {
			return nil, err
		}
		counts = append(counts, rc)
	}
	return counts, rows.Err()
}

func GetWeeklyTrends(ctx context.Context) ([]WeeklyCount, error) {
	rows, err := db.Query(ctx, `
		SELECT to_char(date_trunc('week', created_at), 'YYYY-MM-DD') AS week, COUNT(*)
		FROM referrals
		WHERE created_at >= NOW() - INTERVAL '12 weeks'
		GROUP BY date_trunc('week', created_at)
		ORDER BY week
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weeks []WeeklyCount
	for rows.Next() {
		var wc WeeklyCount
		if err := rows.Scan(&wc.Week, &wc.Count); err != nil {
			return nil, err
		}
		weeks = append(weeks, wc)
	}
	return weeks, rows.Err()
}

// --- Sync log ---

func CreateSyncLog(ctx context.Context, syncType string) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `INSERT INTO sync_log (sync_type, status) VALUES ($1, 'running') RETURNING id`, syncType).Scan(&id)
	return id, err
}

func CompleteSyncLog(ctx context.Context, id int64, count int, errMsg *string) error {
	status := "success"
	if errMsg != nil {
		status = "error"
	}
	_, err := db.Exec(ctx, `UPDATE sync_log SET status = $1, records_synced = $2, error_message = $3, finished_at = NOW() WHERE id = $4`,
		status, count, errMsg, id)
	return err
}

func GetLastSync(ctx context.Context) (*SyncLogEntry, error) {
	row := db.QueryRow(ctx, `SELECT id, sync_type, status, records_synced, error_message, started_at, finished_at FROM sync_log ORDER BY started_at DESC LIMIT 1`)
	var s SyncLogEntry
	err := row.Scan(&s.ID, &s.SyncType, &s.Status, &s.RecordsSynced, &s.ErrorMessage, &s.StartedAt, &s.FinishedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
