package conflict

import (
	"fmt"

	"github.com/agenthub/server/internal/models"
)

// Strategy constants for conflict resolution.
const (
	StrategyLastWriteWins = "last_write_wins"
	StrategyStateMachine  = "state_machine"
	StrategyVersionCheck  = "version_check"
)

// ConflictError represents a version or state conflict between two changes.
type ConflictError struct {
	EntityType     string `json:"entity_type"`
	EntityID       string `json:"entity_id"`
	Reason         string `json:"reason"`
	CurrentVersion int    `json:"current_version,omitempty"`
	IncomingHash   string `json:"incoming_hash,omitempty"`
	ExistingHash   string `json:"existing_hash,omitempty"`
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict on %s/%s: %s", e.EntityType, e.EntityID, e.Reason)
}

// Resolution describes how a conflict was resolved.
type Resolution struct {
	Strategy   string `json:"strategy"`
	Accepted   bool   `json:"accepted"`
	NewVersion int    `json:"new_version,omitempty"`
	Message    string `json:"message"`
}

// Resolver detects and resolves conflicts across different entity types.
// It implements three strategies:
//  1. Artifact: Last-Write-Wins with notification
//  2. Task status: State machine validation
//  3. Context: Version check
type Resolver struct{}

// NewResolver creates a new conflict Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// ResolveArtifact resolves artifact conflicts using a Last-Write-Wins strategy.
// If the incoming content hash differs from the existing hash, the incoming
// version is accepted but a conflict notification is generated.
//
// Parameters:
//   - existingHash: the content hash of the current artifact version
//   - incomingHash: the content hash of the incoming update
//   - currentVersion: the current version number of the artifact
//
// Returns a Resolution indicating the update was accepted and the new version number.
// Returns a ConflictError for informational purposes when hashes differ (LWW still accepts).
func (r *Resolver) ResolveArtifact(entityID, existingHash, incomingHash string, currentVersion int) (*Resolution, *ConflictError) {
	if existingHash == incomingHash {
		// No conflict: content is identical.
		return &Resolution{
			Strategy:   StrategyLastWriteWins,
			Accepted:   true,
			NewVersion: currentVersion,
			Message:    "no change detected",
		}, nil
	}

	// Last-Write-Wins: accept the incoming version but report the conflict.
	conflictErr := &ConflictError{
		EntityType:   "artifact",
		EntityID:     entityID,
		Reason:       "content hash mismatch; incoming version accepted (last-write-wins)",
		IncomingHash: incomingHash,
		ExistingHash: existingHash,
	}

	return &Resolution{
		Strategy:   StrategyLastWriteWins,
		Accepted:   true,
		NewVersion: currentVersion + 1,
		Message:    "conflict detected; incoming version accepted via last-write-wins",
	}, conflictErr
}

// ResolveTaskStatus resolves task status conflicts using state machine validation.
// It checks whether the requested transition from the current status to the new
// status is allowed according to the task state machine.
//
// Returns a Resolution if the transition is valid, or a ConflictError if it is not.
func (r *Resolver) ResolveTaskStatus(taskID string, currentStatus, newStatus string) (*Resolution, error) {
	// Build a temporary task to use the state machine validation.
	task := &models.Task{Status: currentStatus}

	if !task.CanTransitionTo(newStatus) {
		return nil, &ConflictError{
			EntityType: "task",
			EntityID:   taskID,
			Reason:     fmt.Sprintf("invalid status transition: %s -> %s", currentStatus, newStatus),
		}
	}

	return &Resolution{
		Strategy: StrategyStateMachine,
		Accepted: true,
		Message:  fmt.Sprintf("valid transition: %s -> %s", currentStatus, newStatus),
	}, nil
}

// ResolveContext resolves context update conflicts using version checking.
// The update is only accepted if the incoming base version matches the current
// version, preventing lost updates.
//
// Parameters:
//   - entityID: identifier for the context being updated
//   - currentVersion: the current stored version
//   - incomingBaseVersion: the version the updater based their changes on
//   - existingHash: the content hash of the current version
//   - incomingHash: the content hash of the incoming update
//
// Returns a Resolution if versions match, or a ConflictError if they diverge.
func (r *Resolver) ResolveContext(entityID string, currentVersion, incomingBaseVersion int, existingHash, incomingHash string) (*Resolution, error) {
	if currentVersion != incomingBaseVersion {
		return nil, &ConflictError{
			EntityType:     "context",
			EntityID:       entityID,
			Reason:         fmt.Sprintf("version mismatch: current=%d, incoming_base=%d", currentVersion, incomingBaseVersion),
			CurrentVersion: currentVersion,
			ExistingHash:   existingHash,
			IncomingHash:   incomingHash,
		}
	}

	// If the content is identical, no version bump needed.
	if existingHash == incomingHash {
		return &Resolution{
			Strategy:   StrategyVersionCheck,
			Accepted:   true,
			NewVersion: currentVersion,
			Message:    "no change detected",
		}, nil
	}

	return &Resolution{
		Strategy:   StrategyVersionCheck,
		Accepted:   true,
		NewVersion: currentVersion + 1,
		Message:    "version check passed; update accepted",
	}, nil
}
