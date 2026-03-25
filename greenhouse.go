package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
)

const greenhouseBaseURL = "https://harvest.greenhouse.io/v1"

type greenhouseClient struct {
	apiKey string
	client *http.Client
}

var ghClient *greenhouseClient

func initGreenhouseClient() {
	key := os.Getenv("GREENHOUSE_API_KEY")
	if key == "" {
		zap.L().Warn("GREENHOUSE_API_KEY not set, Greenhouse integration disabled")
		return
	}
	ghClient = &greenhouseClient{
		apiKey: key,
		client: &http.Client{},
	}
	zap.L().Info("Greenhouse API client initialized")
}

// --- Greenhouse API response types ---

type GHJob struct {
	ID         int64         `json:"id"`
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	Offices    []GHOffice    `json:"offices"`
	Departments []GHDept     `json:"departments"`
}

type GHOffice struct {
	Name string `json:"name"`
}

type GHDept struct {
	Name string `json:"name"`
}

type GHCandidate struct {
	ID             int64             `json:"id"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	ApplicationIDs []int64           `json:"application_ids"`
	Applications   []GHApplication   `json:"applications"`
}

type GHApplication struct {
	ID              int64               `json:"id"`
	CandidateID     int64               `json:"candidate_id"`
	Status          string              `json:"status"`
	Source          *GHSource           `json:"source"`
	CreditedTo      *GHCreditedTo       `json:"credited_to"`
	CurrentStage    *GHStage            `json:"current_stage"`
	Jobs            []GHApplicationJob  `json:"jobs"`
	RejectedAt      *string             `json:"rejected_at"`
	Prospect        bool                `json:"prospect"`
}

type GHSource struct {
	ID   int64  `json:"id"`
	Name string `json:"public_name"`
}

type GHCreditedTo struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Name      string `json:"name"`
}

type GHStage struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type GHApplicationJob struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// --- API helpers ---

func (c *greenhouseClient) get(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", greenhouseBaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.apiKey, "")

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// getPaginated fetches all pages for an endpoint.
func (c *greenhouseClient) getPaginated(ctx context.Context, endpoint string, params map[string]string) ([]json.RawMessage, error) {
	if params == nil {
		params = map[string]string{}
	}
	params["per_page"] = "500"

	var all []json.RawMessage
	page := 1

	for {
		params["page"] = fmt.Sprintf("%d", page)
		body, err := c.get(ctx, endpoint, params)
		if err != nil {
			return all, err
		}

		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil {
			return all, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		all = append(all, batch...)

		if len(batch) < 500 {
			break
		}
		page++
	}

	return all, nil
}

// FetchOpenJobs returns all open jobs, optionally filtered by department substring.
func FetchOpenJobs(ctx context.Context, departmentFilter string) ([]GHJob, error) {
	if ghClient == nil {
		return nil, fmt.Errorf("greenhouse client not initialized")
	}

	raw, err := ghClient.getPaginated(ctx, "/jobs", map[string]string{"status": "open"})
	if err != nil {
		return nil, err
	}

	var jobs []GHJob
	for _, r := range raw {
		var j GHJob
		if err := json.Unmarshal(r, &j); err != nil {
			zap.L().Warn("failed to parse job", zap.Error(err))
			continue
		}

		if departmentFilter != "" {
			match := false
			for _, d := range j.Departments {
				if strings.Contains(strings.ToLower(d.Name), strings.ToLower(departmentFilter)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
}

// FetchReferralApplications fetches applications credited as referrals.
func FetchReferralApplications(ctx context.Context) ([]GHApplication, error) {
	if ghClient == nil {
		return nil, fmt.Errorf("greenhouse client not initialized")
	}

	raw, err := ghClient.getPaginated(ctx, "/applications", map[string]string{})
	if err != nil {
		return nil, err
	}

	var referrals []GHApplication
	for _, r := range raw {
		var app GHApplication
		if err := json.Unmarshal(r, &app); err != nil {
			zap.L().Warn("failed to parse application", zap.Error(err))
			continue
		}

		if app.Source != nil && strings.Contains(strings.ToLower(app.Source.Name), "referral") {
			referrals = append(referrals, app)
		}
	}

	return referrals, nil
}

// FetchCandidate fetches a single candidate by ID.
func FetchCandidate(ctx context.Context, candidateID int64) (*GHCandidate, error) {
	if ghClient == nil {
		return nil, fmt.Errorf("greenhouse client not initialized")
	}

	body, err := ghClient.get(ctx, fmt.Sprintf("/candidates/%d", candidateID), nil)
	if err != nil {
		return nil, err
	}

	var c GHCandidate
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("failed to parse candidate: %w", err)
	}

	return &c, nil
}
