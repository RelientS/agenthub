package handler

import "errors"

// Handler-level sentinel errors.
var (
	// ErrMissingAgentID is returned when the agent_id is not present in the context.
	ErrMissingAgentID = errors.New("missing agent_id in context")
)
