package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RegisterAPIRoutes(r *gin.Engine) {
	api := r.Group("/api")

	api.GET("/referrals", handleListReferrals())
	api.GET("/referrals/stats", handleReferralStats())
	api.GET("/referrals/by-stage", handleReferralsByStage())
	api.GET("/referrals/by-role", handleReferralsByRole())
	api.GET("/referrals/quarterly", handleQuarterlyTrends())

	api.GET("/jobs", handleListJobs())

	api.GET("/teams", handleGetTeams())
	api.GET("/recruiters", handleGetRecruiters())

	api.GET("/refer/form", handleReferralForm())
	api.GET("/refer/jobs", handleReferralJobs())
	api.POST("/refer", handleSubmitReferral())
}

func handleListReferrals() gin.HandlerFunc {
	return func(c *gin.Context) {
		stage := c.Query("stage")
		role := c.Query("role")

		resp, err := dlListReferrals(c.Request.Context(), stage, role)
		if err != nil {
			zap.L().Error("datalake referral query failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		referrals := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			referrals = append(referrals, gin.H{
				"id":             toStr(row["application_id"]),
				"candidate_name": toStr(row["candidate_name"]),
				"linkedin_url":   toStr(row["linkedin_url"]),
				"role":           toStr(row["role"]),
				"job_id":         toStr(row["job_id"]),
				"referrer_name":  toStr(row["referrer_name"]),
				"stage":          toStr(row["stage"]),
				"app_status":     toStr(row["app_status"]),
				"created_at":     toStr(row["applied_at"]),
				"company":        toStr(row["company"]),
				"current_title":  toStr(row["current_title"]),
				"source":         "datalake",
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "referrals": referrals})
	}
}

func handleReferralStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		statsResp, err := dlReferralStats(c.Request.Context())
		if err != nil {
			zap.L().Error("datalake stats query failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		jobsResp, err := dlListJobs(c.Request.Context())
		if err != nil {
			zap.L().Error("datalake jobs count query failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		row := statsResp.Rows[0]

		stagesResp, _ := dlReferralsByStage(c.Request.Context())
		stages := map[string]int{}
		if stagesResp != nil {
			for _, r := range stagesResp.Rows {
				stages[toStr(r["stage"])] = toInt(r["count"])
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"stats": gin.H{
				"total_referrals": toInt(row["total_referrals"]),
				"active":          toInt(row["active"]),
				"rejected":        toInt(row["rejected"]),
				"open_jobs":       len(jobsResp.Rows),
				"priority_jobs":   0,
				"stages":          stages,
			},
		})
	}
}

func handleReferralsByStage() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := dlReferralsByStage(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		stages := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			stages = append(stages, gin.H{
				"stage": toStr(row["stage"]),
				"count": toInt(row["count"]),
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "stages": stages})
	}
}

func handleReferralsByRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := dlReferralsByRole(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		roles := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			roles = append(roles, gin.H{
				"role":  toStr(row["role"]),
				"count": toInt(row["count"]),
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "roles": roles})
	}
}

func handleQuarterlyTrends() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := dlQuarterlyTrends(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		quarters := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			quarters = append(quarters, gin.H{
				"quarter": toStr(row["quarter"]),
				"count":   toInt(row["count"]),
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "quarters": quarters})
	}
}

func handleListJobs() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := dlJobsWithReferralCounts(c.Request.Context())
		if err != nil {
			zap.L().Error("datalake jobs query failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		jobs := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			title := toStr(row["title"])
			var jobURL *string
			if ghClient != nil && ghClient.apiKey != "" {
				if u := ghClient.JobURL(c.Request.Context(), title); u != "" {
					jobURL = &u
				}
			}
			jobs = append(jobs, gin.H{
				"id":             toStr(row["id"]),
				"title":          title,
				"status":         toStr(row["status"]),
				"department":     toStr(row["department_id"]),
				"location":       toStr(row["location_id"]),
				"referral_count": toInt(row["referral_count"]),
				"job_url":        jobURL,
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "jobs": jobs})
	}
}

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

func handleReferralForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		schema := gin.H{
			"success": true,
			"fields": []gin.H{
				{"path": "referrer_email", "title": "Your Work Email", "type": "email", "required": true},
				{"path": "candidate_name", "title": "Candidate Full Name", "type": "text", "required": true},
				{"path": "candidate_email", "title": "Candidate Email", "type": "email", "required": true},
				{"path": "linkedin_url", "title": "LinkedIn URL", "type": "url", "required": false},
				{"path": "phone", "title": "Phone Number", "type": "tel", "required": false},
				{"path": "job_id", "title": "Job", "type": "select", "required": true},
				{"path": "relationship", "title": "How do you know this person?", "type": "select", "required": false},
				{"path": "note", "title": "Why are you recommending them?", "type": "textarea", "required": false},
			},
		}
		c.JSON(http.StatusOK, schema)
	}
}

func handleReferralJobs() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := dlListJobs(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		jobs := make([]gin.H, 0, len(resp.Rows))
		for _, row := range resp.Rows {
			jobs = append(jobs, gin.H{
				"id":    toStr(row["id"]),
				"title": toStr(row["title"]),
			})
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "jobs": jobs})
	}
}

func handleSubmitReferral() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ghClient == nil || ghClient.apiKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Greenhouse API not configured. Set GREENHOUSE_API_KEY to enable referral submissions."})
			return
		}

		var sub referralSubmission
		if err := c.ShouldBindJSON(&sub); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}

		if sub.ReferrerEmail == "" || sub.CandidateName == "" || sub.CandidateEmail == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referrer_email, candidate_name, and candidate_email are required"})
			return
		}

		raw, err := ghClient.submitReferral(c.Request.Context(), sub)
		if err != nil {
			zap.L().Error("greenhouse referral submission failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", raw)
	}
}
