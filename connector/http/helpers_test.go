package http

import (
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// connectorStep is a test helper to create a connector.Step.
func connectorStep(action string, params map[string]any) connector.Step {
	return connector.Step{
		Action:     action,
		Parameters: params,
	}
}

// connectorStepWithTimeout is a test helper to create a connector.Step with a timeout.
func connectorStepWithTimeout(action string, params map[string]any, timeout time.Duration) connector.Step {
	return connector.Step{
		Action:     action,
		Parameters: params,
		Timeout:    timeout,
	}
}
