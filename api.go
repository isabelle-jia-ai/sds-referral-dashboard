package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.apps.applied.dev/lib/slacklib"
	"go.uber.org/zap"
)

func RegisterAPIRoutes(r *gin.Engine, bot *slacklib.Bot) {
	api := r.Group("/api")

	// Referrals
	api.GET("/referrals", handleListReferrals())
	api.GET("/referrals/stats", handleReferralStats())
	api.GET("/referrals/by-stage", handleReferralsByStage())
	api.GET("/referrals/by-role", handleReferralsByRole())
	api.GET("/referrals/weekly", handleWeeklyTrends())

	// Jobs
	api.GET("/jobs", handleListJobs())
	api.PATCH("/jobs/:id/priority", handleTogglePriority())

	// Teams / recruiters
	api.GET("/teams", handleGetTeams())
	api.GET("/recruiters", handleGetRecruiters())

	// Sync
	api.GET("/sync/status", handleSyncStatus())
	api.POST("/sync/trigger", handleTriggerSync())
}

// --- Referral handlers ---

func handleListReferrals() gin.HandlerFunc {
	return func(c *gin.Context) {
		stage := c.Query("stage")
		role := c.Query("role")
		var jobID *int64
		if j := c.Query("job_id"); j != "" {
			if id, err := strconv.ParseInt(j, 10, 64); err == nil {
				jobID = &id
			}
		}

		referrals, err := ListReferrals(c.Request.Context(), stage, role, jobID)
		if err != nil {
			zap.L().Error("failed to list referrals", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if referrals == nil {
			referrals = []Referral{}
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "referrals": referrals})
	}
}

func handleReferralStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := GetReferralStats(c.Request.Context())
		if err != nil {
			zap.L().Error("failed to get stats", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "stats": stats})
	}
}

func handleReferralsByStage() gin.HandlerFunc {
	return func(c *gin.Context) {
		stages, err := GetReferralsByStage(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "stages": stages})
	}
}

func handleReferralsByRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, err := GetReferralsByRole(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "roles": roles})
	}
}

func handleWeeklyTrends() gin.HandlerFunc {
	return func(c *gin.Context) {
		weeks, err := GetWeeklyTrends(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "weeks": weeks})
	}
}

// --- Job handlers ---

func handleListJobs() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "open")
		priorityOnly := c.Query("priority") == "true"

		jobs, err := ListJobs(c.Request.Context(), status, priorityOnly)
		if err != nil {
			zap.L().Error("failed to list jobs", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if jobs == nil {
			jobs = []Job{}
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "jobs": jobs})
	}
}

func handleTogglePriority() gin.HandlerFunc {
	return func(c *gin.Context) {
		ghID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
			return
		}

		var req struct {
			IsPriority bool `json:"is_priority"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := SetJobPriority(c.Request.Context(), ghID, req.IsPriority); err != nil {
			zap.L().Error("failed to set priority", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// --- Team / recruiter handlers ---

func handleGetTeams() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "teams": GetTeamNames()})
	}
}

func handleGetRecruiters() gin.HandlerFunc {
	return func(c *gin.Context) {
		recruiters := GetAllRecruiters()
		result := make([]gin.H, len(recruiters))
		for i, r := range recruiters {
			result[i] = gin.H{"name": r.RecruiterName, "email": r.RecruiterEmail, "team": r.Team}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "recruiters": result})
	}
}

// --- Sync handlers ---

func handleSyncStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		last, err := GetLastSync(c.Request.Context())
		if err != nil {
			zap.L().Error("failed to get sync status", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "last_sync": last})
	}
}

func handleTriggerSync() gin.HandlerFunc {
	return func(c *gin.Context) {
		go RunFullSync(context.Background())
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "sync started"})
	}
}
