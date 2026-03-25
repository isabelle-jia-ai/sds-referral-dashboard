package main

import (
	"os"
	"regexp"
	"strings"

	"go.apps.applied.dev/lib/slacklib"
	"go.uber.org/zap"
)

// ReferralChannelID is the Slack channel to monitor for referral messages.
// Set via REFERRAL_CHANNEL_ID env var, falls back to a default.
var ReferralChannelID string

func init() {
	ReferralChannelID = os.Getenv("REFERRAL_CHANNEL_ID")
	if ReferralChannelID == "" {
		ReferralChannelID = "C0AMGLG0S74"
	}
}

var linkedinPattern = regexp.MustCompile(`https?://(?:www\.)?linkedin\.com/in/[^\s>]+`)

// RegisterSlackMonitor sets up passive listeners on the referral channel.
// This is read-only — we capture referral messages but do not reply.
func RegisterSlackMonitor(bot *slacklib.Bot) {
	bot.OnMessage(func(ctx *slacklib.MessageContext) {
		if ctx.ChannelID != ReferralChannelID {
			return
		}

		if ctx.BotID != "" || ctx.ThreadTS != "" {
			return
		}

		text := ctx.Text
		if text == "" {
			return
		}

		candidateName := extractCandidateName(text)
		if candidateName == "" {
			return
		}

		linkedin := ""
		if m := linkedinPattern.FindString(text); m != "" {
			linkedin = m
		}

		role := extractRole(text)
		userID := ctx.UserID
		channelID := ctx.ChannelID
		eventTS := ctx.EventTS

		ref := &Referral{
			CandidateName:   candidateName,
			LinkedInURL:     strPtr(linkedin),
			Role:            strPtr(role),
			ReferrerSlackID: &userID,
			Stage:           "submitted",
			SlackChannelID:  &channelID,
			SlackMessageTS:  &eventTS,
			Source:          "slack",
		}

		if err := UpsertReferral(ctx.Context(), ref); err != nil {
			zap.L().Error("failed to save referral from Slack",
				zap.String("user", ctx.UserID),
				zap.Error(err))
			return
		}

		zap.L().Info("captured referral from Slack",
			zap.String("candidate", candidateName),
			zap.String("user", ctx.UserID))
	})
}

// extractCandidateName tries to pull a candidate name from the message text.
// Looks for patterns like "Referral: John Doe" or "referring John Doe" etc.
func extractCandidateName(text string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)referr(?:al|ing)[:\s]+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)`),
		regexp.MustCompile(`(?i)candidate[:\s]+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)`),
		regexp.MustCompile(`(?i)name[:\s]+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)`),
		regexp.MustCompile(`(?i)submitted?\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)`),
	}

	for _, p := range patterns {
		if m := p.FindStringSubmatch(text); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}

	return ""
}

// extractRole tries to pull a role/position from the message text.
func extractRole(text string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:role|position|for)[:\s]+(.+?)(?:\n|$)`),
		regexp.MustCompile(`(?i)(?:hiring team|team)[:\s]+(.+?)(?:\n|$)`),
	}

	for _, p := range patterns {
		if m := p.FindStringSubmatch(text); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}

	return ""
}
