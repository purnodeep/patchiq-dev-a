package v1

import (
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// NewDeploymentHandlerForTest creates a DeploymentHandler without pool and riverClient.
// Only use in unit tests where Create's transaction path is not exercised.
func NewDeploymentHandlerForTest(q DeploymentQuerier, eventBus domain.EventBus, evaluator *deployment.Evaluator, sm *deployment.StateMachine) *DeploymentHandler {
	return &DeploymentHandler{q: q, eventBus: eventBus, evaluator: evaluator, sm: sm}
}

// NewDeploymentHandlerWithCancelTxForTest creates a DeploymentHandler with a
// CancelTxFactory for unit-testing Cancel's post-commit event emission.
func NewDeploymentHandlerWithCancelTxForTest(q DeploymentQuerier, cancelTxFactory CancelTxFactory, eventBus domain.EventBus, evaluator *deployment.Evaluator, sm *deployment.StateMachine) *DeploymentHandler {
	return &DeploymentHandler{q: q, cancelTxFactory: cancelTxFactory, eventBus: eventBus, evaluator: evaluator, sm: sm}
}
