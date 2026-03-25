package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// StartPeriodicSync runs Greenhouse sync on a timer.
func StartPeriodicSync(ctx context.Context, intervalMinutes int) {
	if ghClient == nil {
		zap.L().Warn("greenhouse client not available, periodic sync disabled")
		return
	}

	// Run an initial sync on startup
	RunFullSync(ctx)

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			RunFullSync(ctx)
		}
	}
}

// RunFullSync performs a complete sync of jobs and referral applications from Greenhouse.
func RunFullSync(ctx context.Context) {
	if db == nil {
		zap.L().Warn("database not available, skipping sync")
		return
	}

	zap.L().Info("starting Greenhouse sync")

	logID, err := CreateSyncLog(ctx, "full")
	if err != nil {
		zap.L().Error("failed to create sync log", zap.Error(err))
	}

	totalSynced := 0

	jobCount, err := syncJobs(ctx)
	if err != nil {
		zap.L().Error("job sync failed", zap.Error(err))
		errMsg := err.Error()
		CompleteSyncLog(ctx, logID, totalSynced, &errMsg)
		return
	}
	totalSynced += jobCount

	refCount, err := syncReferralApplications(ctx)
	if err != nil {
		zap.L().Error("referral sync failed", zap.Error(err))
		errMsg := err.Error()
		CompleteSyncLog(ctx, logID, totalSynced, &errMsg)
		return
	}
	totalSynced += refCount

	CompleteSyncLog(ctx, logID, totalSynced, nil)
	zap.L().Info("Greenhouse sync complete", zap.Int("jobs", jobCount), zap.Int("referrals", refCount))
}

func syncJobs(ctx context.Context) (int, error) {
	jobs, err := FetchOpenJobs(ctx, "")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch jobs: %w", err)
	}

	activeIDs := make([]int64, 0, len(jobs))

	for _, ghJob := range jobs {
		dept := ""
		if len(ghJob.Departments) > 0 {
			dept = ghJob.Departments[0].Name
		}

		loc := ""
		locs := make([]string, 0, len(ghJob.Offices))
		for _, o := range ghJob.Offices {
			locs = append(locs, o.Name)
		}
		if len(locs) > 0 {
			loc = strings.Join(locs, ", ")
		}

		j := &Job{
			GreenhouseID: ghJob.ID,
			Title:        ghJob.Name,
			Department:   strPtr(dept),
			Team:         strPtr(dept),
			Status:       "open",
			Location:     strPtr(loc),
		}

		if err := UpsertJob(ctx, j); err != nil {
			zap.L().Warn("failed to upsert job", zap.Int64("id", ghJob.ID), zap.Error(err))
			continue
		}

		activeIDs = append(activeIDs, ghJob.ID)
	}

	if err := MarkClosedJobsMissing(ctx, activeIDs); err != nil {
		zap.L().Warn("failed to mark closed jobs", zap.Error(err))
	}

	return len(activeIDs), nil
}

func syncReferralApplications(ctx context.Context) (int, error) {
	apps, err := FetchReferralApplications(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch referral applications: %w", err)
	}

	count := 0
	for _, app := range apps {
		stage := "submitted"
		if app.CurrentStage != nil {
			stage = app.CurrentStage.Name
		}
		if app.RejectedAt != nil {
			stage = "rejected"
		}

		var jobID *int64
		roleName := ""
		if len(app.Jobs) > 0 {
			jobID = &app.Jobs[0].ID
			roleName = app.Jobs[0].Name
		}

		referrerName := ""
		if app.CreditedTo != nil {
			referrerName = app.CreditedTo.Name
			if referrerName == "" {
				referrerName = app.CreditedTo.FirstName + " " + app.CreditedTo.LastName
			}
		}

		candidate, err := FetchCandidate(ctx, app.CandidateID)
		if err != nil {
			zap.L().Warn("failed to fetch candidate for application", zap.Int64("app_id", app.ID), zap.Error(err))
			continue
		}

		candidateName := strings.TrimSpace(candidate.FirstName + " " + candidate.LastName)

		ref := &Referral{
			CandidateName:           candidateName,
			Role:                    strPtr(roleName),
			JobGreenhouseID:         jobID,
			ReferrerName:            strPtr(referrerName),
			Stage:                   stage,
			GreenhouseCandidateID:   &app.CandidateID,
			GreenhouseApplicationID: &app.ID,
			Source:                  "greenhouse",
		}

		if err := UpsertReferralByGHCandidate(ctx, ref); err != nil {
			zap.L().Warn("failed to upsert referral", zap.Int64("app_id", app.ID), zap.Error(err))
			continue
		}
		count++
	}

	// Update referral counts on jobs
	if db != nil {
		_, err := db.Exec(ctx, `
			UPDATE jobs SET referral_count = sub.cnt, updated_at = NOW()
			FROM (SELECT job_greenhouse_id, COUNT(*) AS cnt FROM referrals WHERE job_greenhouse_id IS NOT NULL GROUP BY job_greenhouse_id) sub
			WHERE jobs.greenhouse_id = sub.job_greenhouse_id
		`)
		if err != nil {
			zap.L().Warn("failed to update job referral counts", zap.Error(err))
		}
	}

	return count, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
