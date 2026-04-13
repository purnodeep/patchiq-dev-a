package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// SendJobArgs is the payload for notification send River jobs.
type SendJobArgs struct {
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	TriggerType string `json:"trigger_type"`
	ChannelID   string `json:"channel_id"`
	ShoutrrrURL string `json:"shoutrrr_url"`
	Message     string `json:"message"`
	ChannelType string `json:"channel_type"` // "email", "slack", or "webhook"
	Recipient   string `json:"recipient"`    // email address or Slack channel
	Subject     string `json:"subject"`      // notification subject line
}

func (SendJobArgs) Kind() string { return "notification_send" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SendJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// HistoryRecord represents a notification history entry to record.
type HistoryRecord struct {
	ID           string
	TenantID     string
	TriggerType  string
	ChannelID    string
	UserID       string
	Status       string
	Payload      json.RawMessage
	ErrorMessage string
	ChannelType  string // NEW
	Recipient    string // NEW
	Subject      string // NEW
}

// HistoryRecorder persists notification history entries.
type HistoryRecorder interface {
	Record(ctx context.Context, rec HistoryRecord) error
}

// SendWorker processes notification send jobs.
type SendWorker struct {
	river.WorkerDefaults[SendJobArgs]
	sender   Sender
	recorder HistoryRecorder
	eventBus domain.EventBus
}

func NewSendWorker(sender Sender, recorder HistoryRecorder, eventBus domain.EventBus) *SendWorker {
	return &SendWorker{sender: sender, recorder: recorder, eventBus: eventBus}
}

func (w *SendWorker) Work(ctx context.Context, job *river.Job[SendJobArgs]) error {
	args := job.Args

	sendErr := w.sender.Send(ctx, args.ShoutrrrURL, args.Message)

	status := "sent"
	var errMsg string
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
	}

	payloadJSON, marshalErr := json.Marshal(map[string]string{
		"message":      args.Message,
		"trigger_type": args.TriggerType,
	})
	if marshalErr != nil {
		slog.ErrorContext(ctx, "marshal notification payload failed",
			"tenant_id", args.TenantID, "trigger_type", args.TriggerType, "error", marshalErr)
		payloadJSON = json.RawMessage("{}")
	}

	rec := HistoryRecord{
		ID:           domain.NewEventID(),
		TenantID:     args.TenantID,
		TriggerType:  args.TriggerType,
		ChannelID:    args.ChannelID,
		UserID:       args.UserID,
		Status:       status,
		Payload:      payloadJSON,
		ErrorMessage: errMsg,
		ChannelType:  args.ChannelType,
		Recipient:    args.Recipient,
		Subject:      args.Subject,
	}
	if recErr := w.recorder.Record(ctx, rec); recErr != nil {
		slog.ErrorContext(ctx, "record notification history failed",
			"tenant_id", args.TenantID, "trigger_type", args.TriggerType, "error", recErr)
	}

	// Emit domain event (best-effort)
	if w.eventBus != nil {
		eventType := "notification.sent"
		if sendErr != nil {
			eventType = "notification.failed"
		}
		evt := domain.DomainEvent{
			ID:         domain.NewEventID(),
			Type:       eventType,
			TenantID:   args.TenantID,
			ActorID:    "system",
			ActorType:  domain.ActorSystem,
			Resource:   "notification",
			ResourceID: rec.ID,
			Action:     eventType,
			Payload:    map[string]string{"trigger_type": args.TriggerType, "channel_id": args.ChannelID, "status": status},
			Timestamp:  time.Now(),
		}
		if emitErr := w.eventBus.Emit(ctx, evt); emitErr != nil {
			slog.ErrorContext(ctx, "emit notification event failed",
				"event_type", eventType, "error", emitErr)
		}
	}

	if sendErr != nil {
		return fmt.Errorf("send notification via %s: %w", args.TriggerType, sendErr)
	}
	return nil
}
