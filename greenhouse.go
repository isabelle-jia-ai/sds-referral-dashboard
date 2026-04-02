package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	ghBoardAPI   = "https://boards-api.greenhouse.io/v1/boards/appliedintuition/jobs"
	ghHarvestAPI = "https://harvest.greenhouse.io/v1"
	ghSourceID   = 4000077005
)

type greenhouseClient struct {
	apiKey string
	http   *http.Client

	mu          sync.RWMutex
	titleURLMap map[string]string // normalised title -> absolute_url
	titleIDMap  map[string]int    // normalised title -> greenhouse numeric job id
	cachedAt    time.Time
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
		http:   &http.Client{Timeout: 30 * time.Second},
	}
	zap.L().Info("greenhouse client initialized")
}

// Board API types (public, no auth)

type ghBoardResponse struct {
	Jobs []ghBoardJob `json:"jobs"`
}

type ghBoardJob struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	AbsoluteURL string `json:"absolute_url"`
}

func normaliseTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (c *greenhouseClient) refreshBoardCache(ctx context.Context) error {
	c.mu.RLock()
	if time.Since(c.cachedAt) < time.Hour && len(c.titleURLMap) > 0 {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", ghBoardAPI, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("greenhouse board API %d: %s", resp.StatusCode, string(body))
	}

	var board ghBoardResponse
	if err := json.Unmarshal(body, &board); err != nil {
		return err
	}

	urlMap := make(map[string]string, len(board.Jobs))
	idMap := make(map[string]int, len(board.Jobs))
	for _, j := range board.Jobs {
		key := normaliseTitle(j.Title)
		urlMap[key] = j.AbsoluteURL
		idMap[key] = j.ID
	}

	c.mu.Lock()
	c.titleURLMap = urlMap
	c.titleIDMap = idMap
	c.cachedAt = time.Now()
	c.mu.Unlock()

	zap.L().Info("greenhouse board cache refreshed", zap.Int("jobs", len(board.Jobs)))
	return nil
}

// JobURL returns the Greenhouse board URL for a given job title, or "" if not found.
func (c *greenhouseClient) JobURL(ctx context.Context, title string) string {
	if err := c.refreshBoardCache(ctx); err != nil {
		zap.L().Warn("board cache refresh failed", zap.Error(err))
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if u, ok := c.titleURLMap[normaliseTitle(title)]; ok {
		return u
	}

	norm := normaliseTitle(title)
	for key, u := range c.titleURLMap {
		if strings.Contains(key, norm) || strings.Contains(norm, key) {
			return u
		}
	}
	return ""
}

// lookupGHJobID returns the Greenhouse numeric job ID for a given Ashby job title.
func (c *greenhouseClient) lookupGHJobID(ctx context.Context, title string) (int, error) {
	if err := c.refreshBoardCache(ctx); err != nil {
		return 0, fmt.Errorf("board cache refresh failed: %w", err)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if id, ok := c.titleIDMap[normaliseTitle(title)]; ok {
		return id, nil
	}

	norm := normaliseTitle(title)
	for key, id := range c.titleIDMap {
		if strings.Contains(key, norm) || strings.Contains(norm, key) {
			return id, nil
		}
	}
	return 0, fmt.Errorf("no greenhouse job found matching %q", title)
}

// Harvest API helpers (authenticated)

func (c *greenhouseClient) harvestGet(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ghHarvestAPI+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.apiKey, "")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("greenhouse harvest %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *greenhouseClient) harvestPost(ctx context.Context, path string, payload interface{}, onBehalfOf string) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ghHarvestAPI+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, "")
	if onBehalfOf != "" {
		req.Header.Set("On-Behalf-Of", onBehalfOf)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("greenhouse harvest %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

type ghUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"primary_email_address"`
}

// lookupUserByEmail finds a Greenhouse user ID by email address.
func (c *greenhouseClient) lookupUserByEmail(ctx context.Context, email string) (string, error) {
	raw, err := c.harvestGet(ctx, "/users?email="+email)
	if err != nil {
		return "", err
	}

	var users []ghUser
	if err := json.Unmarshal(raw, &users); err != nil {
		return "", fmt.Errorf("parse users response: %w", err)
	}
	if len(users) == 0 {
		return "", fmt.Errorf("no greenhouse user found for %s", email)
	}
	return fmt.Sprintf("%d", users[0].ID), nil
}

type referralSubmission struct {
	ReferrerEmail  string `json:"referrer_email"`
	CandidateName  string `json:"candidate_name"`
	CandidateEmail string `json:"candidate_email"`
	LinkedInURL    string `json:"linkedin_url"`
	Phone          string `json:"phone"`
	JobID          string `json:"job_id"`
	JobTitle       string `json:"job_title"`
	Relationship   string `json:"relationship"`
	Note           string `json:"note"`
}

// submitReferral creates a candidate+application in Greenhouse via Harvest API.
func (c *greenhouseClient) submitReferral(ctx context.Context, sub referralSubmission) ([]byte, error) {
	userID, err := c.lookupUserByEmail(ctx, sub.ReferrerEmail)
	if err != nil {
		return nil, fmt.Errorf("referrer lookup failed: %w", err)
	}

	ghJobID := 0
	if sub.JobTitle != "" {
		ghJobID, _ = c.lookupGHJobID(ctx, sub.JobTitle)
	}
	if ghJobID == 0 && sub.JobID != "" {
		ghJobID, _ = c.lookupGHJobID(ctx, sub.JobID)
	}

	nameParts := strings.SplitN(sub.CandidateName, " ", 2)
	firstName := nameParts[0]
	lastName := ""
	if len(nameParts) > 1 {
		lastName = nameParts[1]
	}

	emails := []map[string]string{
		{"value": sub.CandidateEmail, "type": "personal"},
	}

	candidate := map[string]interface{}{
		"first_name":      firstName,
		"last_name":       lastName,
		"email_addresses": emails,
	}

	if sub.LinkedInURL != "" {
		candidate["social_media_addresses"] = []map[string]string{
			{"value": sub.LinkedInURL},
		}
	}
	if sub.Phone != "" {
		candidate["phone_numbers"] = []map[string]string{
			{"value": sub.Phone, "type": "mobile"},
		}
	}

	app := map[string]interface{}{
		"source_id": ghSourceID,
		"referrer":  map[string]string{"type": "email", "value": sub.ReferrerEmail},
	}
	if ghJobID > 0 {
		app["job_id"] = ghJobID
	}
	candidate["applications"] = []interface{}{app}

	noteBody := ""
	if sub.Relationship != "" {
		noteBody += "Relationship: " + sub.Relationship + "\n"
	}
	if sub.Note != "" {
		noteBody += sub.Note
	}
	if noteBody != "" {
		candidate["activity_feed_notes"] = []map[string]string{
			{"body": strings.TrimSpace(noteBody), "visibility": "public"},
		}
	}

	raw, err := c.harvestPost(ctx, "/candidates", candidate, userID)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
