package v1

import "github.com/skenzeriq/patchiq/internal/shared/domain"

// NewScheduleHandlerForTest creates a ScheduleHandler for unit tests.
func NewScheduleHandlerForTest(q ScheduleQuerier, eventBus domain.EventBus) *ScheduleHandler {
	return &ScheduleHandler{q: q, eventBus: eventBus}
}
