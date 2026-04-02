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
	ghHarvestAPI  = "https://harvest.greenhouse.io/v1"
	ghInternalURL = "https://app.greenhouse.io/sdash"
	ghSourceID    = 4000077005
)

type greenhouseClient struct {
	apiKey string
	http   *http.Client

	mu          sync.RWMutex
	titleURLMap map[string]string // normalised title -> absolute_url
	titleIDMap  map[string]int    // normalised title -> greenhouse numeric job id
	cachedAt    time.Time

	linkedInMu    sync.RWMutex
	linkedInCache map[string]string // email -> linkedin URL (empty string = looked up, none found)
}

var ghClient *greenhouseClient

func initGreenhouseClient() {
	ghClient = &greenhouseClient{
		http: &http.Client{Timeout: 30 * time.Second},
	}
	key := os.Getenv("GREENHOUSE_API_KEY")
	if key != "" {
		ghClient.apiKey = key
		zap.L().Info("greenhouse client initialized (board + harvest)")
	} else {
		zap.L().Info("greenhouse client initialized (board only, referral submission disabled)")
	}
}

type ghHarvestJob struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func normaliseTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// refreshJobCache fetches all jobs from the Greenhouse Harvest API (paginated)
// and builds title-to-URL and title-to-ID maps. Requires API key.
func (c *greenhouseClient) refreshJobCache(ctx context.Context) error {
	if c.apiKey == "" {
		return fmt.Errorf("greenhouse API key not set")
	}

	c.mu.RLock()
	if time.Since(c.cachedAt) < time.Hour && len(c.titleURLMap) > 0 {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	urlMap := make(map[string]string)
	idMap := make(map[string]int)

	for page := 1; ; page++ {
		raw, err := c.harvestGet(ctx, fmt.Sprintf("/jobs?per_page=500&page=%d", page))
		if err != nil {
			return fmt.Errorf("harvest jobs page %d: %w", page, err)
		}

		var jobs []ghHarvestJob
		if err := json.Unmarshal(raw, &jobs); err != nil {
			return fmt.Errorf("parse jobs page %d: %w", page, err)
		}

		for _, j := range jobs {
			key := normaliseTitle(j.Name)
			urlMap[key] = fmt.Sprintf("%s/%d", ghInternalURL, j.ID)
			idMap[key] = j.ID
		}

		if len(jobs) < 500 {
			break
		}
	}

	c.mu.Lock()
	c.titleURLMap = urlMap
	c.titleIDMap = idMap
	c.cachedAt = time.Now()
	c.mu.Unlock()

	zap.L().Info("greenhouse harvest job cache refreshed", zap.Int("jobs", len(urlMap)))
	return nil
}

// JobURL returns the internal Greenhouse job dashboard URL for a given title, or "" if not found.
func (c *greenhouseClient) JobURL(ctx context.Context, title string) string {
	if err := c.refreshJobCache(ctx); err != nil {
		zap.L().Warn("harvest job cache refresh failed", zap.Error(err))
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if u, ok := c.titleURLMap[normaliseTitle(title)]; ok {
		return u
	}
	return ""
}

// lookupGHJobID returns the Greenhouse numeric job ID for a given job title.
func (c *greenhouseClient) lookupGHJobID(ctx context.Context, title string) (int, error) {
	if err := c.refreshJobCache(ctx); err != nil {
		return 0, fmt.Errorf("harvest job cache refresh failed: %w", err)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if id, ok := c.titleIDMap[normaliseTitle(title)]; ok {
		return id, nil
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

// LookupLinkedIn searches Greenhouse for a candidate by email and returns their
// LinkedIn URL from social_media_addresses. Results are cached indefinitely.
func (c *greenhouseClient) LookupLinkedIn(ctx context.Context, email string) string {
	if c.apiKey == "" || email == "" {
		return ""
	}

	c.linkedInMu.RLock()
	if url, ok := c.linkedInCache[email]; ok {
		c.linkedInMu.RUnlock()
		return url
	}
	c.linkedInMu.RUnlock()

	raw, err := c.harvestGet(ctx, "/candidates?email="+email)
	if err != nil {
		zap.L().Debug("linkedin lookup failed", zap.String("email", email), zap.Error(err))
		return ""
	}

	var candidates []struct {
		SocialMedia []struct {
			Value string `json:"value"`
		} `json:"social_media_addresses"`
	}
	if err := json.Unmarshal(raw, &candidates); err != nil {
		return ""
	}

	linkedIn := ""
	for _, cand := range candidates {
		for _, sm := range cand.SocialMedia {
			if strings.Contains(sm.Value, "linkedin.com") {
				linkedIn = sm.Value
				break
			}
		}
		if linkedIn != "" {
			break
		}
	}

	c.linkedInMu.Lock()
	if c.linkedInCache == nil {
		c.linkedInCache = make(map[string]string)
	}
	c.linkedInCache[email] = linkedIn
	c.linkedInMu.Unlock()

	return linkedIn
}

// BulkLookupLinkedIn looks up LinkedIn URLs for a batch of emails concurrently
// with bounded parallelism to respect rate limits.
func (c *greenhouseClient) BulkLookupLinkedIn(ctx context.Context, emails []string) map[string]string {
	result := make(map[string]string, len(emails))
	if c.apiKey == "" {
		return result
	}

	var uncached []string
	c.linkedInMu.RLock()
	for _, email := range emails {
		if url, ok := c.linkedInCache[email]; ok {
			result[email] = url
		} else if email != "" {
			uncached = append(uncached, email)
		}
	}
	c.linkedInMu.RUnlock()

	if len(uncached) == 0 {
		return result
	}

	sem := make(chan struct{}, 5)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, email := range uncached {
		wg.Add(1)
		go func(e string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			url := c.LookupLinkedIn(ctx, e)
			mu.Lock()
			result[e] = url
			mu.Unlock()
		}(email)
	}
	wg.Wait()

	return result
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
