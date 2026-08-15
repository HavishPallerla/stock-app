// Package alerts fans a new trading signal out to users whose watchlist and
// alert preferences match it, via SMS.
package alerts

import (
	"context"
	"fmt"
	"log/slog"

	"stockapp/backend-go/internal/models"
	"stockapp/backend-go/internal/supa"
	"stockapp/backend-go/internal/twilio"
)

type Dispatcher struct {
	supa       *supa.Client
	twilio     *twilio.Client // nil when SMS alerts are disabled
	smsEnabled bool
}

func NewDispatcher(supaClient *supa.Client, twilioClient *twilio.Client, smsEnabled bool) *Dispatcher {
	return &Dispatcher{supa: supaClient, twilio: twilioClient, smsEnabled: smsEnabled}
}

// Dispatch looks up every user whose watchlist/preferences match this
// analysis and sends (or, behind the feature flag, logs-but-skips) an SMS to
// each, logging delivery status to sms_log either way.
func (d *Dispatcher) Dispatch(ctx context.Context, analysis models.TweetAnalysis) error {
	recipients, err := d.supa.MatchingRecipients(ctx, analysis.Ticker, analysis.Signal)
	if err != nil {
		return fmt.Errorf("lookup recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil
	}

	body := formatMessage(analysis)

	for _, r := range recipients {
		logRow := models.SMSLog{
			UserID:          r.UserID,
			TweetAnalysisID: analysis.ID,
		}

		if !d.smsEnabled {
			logRow.Status = "skipped"
			logRow.ErrorMessage = "FEATURE_SMS_ALERTS disabled"
		} else {
			_, sendErr := d.twilio.SendSMS(ctx, r.Phone, body)
			if sendErr != nil {
				logRow.Status = "failed"
				logRow.ErrorMessage = sendErr.Error()
				slog.Warn("alerts: sms send failed", "user_id", r.UserID, "error", sendErr)
			} else {
				logRow.Status = "sent"
			}
		}

		if err := d.supa.InsertSMSLog(ctx, logRow); err != nil {
			slog.Warn("alerts: failed to write sms_log", "user_id", r.UserID, "error", err)
		}
	}

	return nil
}

func formatMessage(a models.TweetAnalysis) string {
	return fmt.Sprintf("[%s] %s (%.0f%% confidence): %s",
		a.Ticker, signalLabel(a.Signal), a.Confidence*100, a.Justification)
}

func signalLabel(signal string) string {
	switch signal {
	case "buy":
		return "BUY"
	case "sell":
		return "SELL"
	default:
		return "HOLD"
	}
}
