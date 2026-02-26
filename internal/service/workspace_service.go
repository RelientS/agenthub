package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// alphanumeric characters used for invite code generation.
const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// ---------- request / response types ----------

// CreateWorkspaceInput holds the input for creating a workspace.
type CreateWorkspaceInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateWorkspaceResponse is returned after successfully creating a workspace.
type CreateWorkspaceResponse struct {
	Workspace *models.Workspace `json:"workspace"`
	AgentID   uuid.UUID         `json:"agent_id"`
	Token     string            `json:"token"`
}

// UpdateWorkspaceInput holds the input for updating a workspace.
type UpdateWorkspaceInput struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// JoinWorkspaceInput holds the input for joining a workspace.
type JoinWorkspaceInput struct {
	InviteCode   string   `json:"invite_code"`
	AgentName    string   `json:"agent_name"`
	AgentRole    string   `json:"agent_role"`
	Capabilities []string `json:"capabilities"`
}

// JoinWorkspaceResponse is returned after successfully joining a workspace.
type JoinWorkspaceResponse struct {
	Workspace *models.Workspace `json:"workspace"`
	Agent     *models.Agent     `json:"agent"`
	Token     string            `json:"token"`
}

// HeartbeatInput holds the input for an agent heartbeat.
type HeartbeatInput struct {
	Status   string                 `json:"status"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ---------- service ----------

// WorkspaceService handles workspace business logic.
type WorkspaceService struct {
	workspaceRepo *repository.WorkspaceRepository
	agentRepo     *repository.AgentRepository
	eventBus      *events.Bus
	jwtSecret     []byte
	jwtExpiry     time.Duration
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(
	wr *repository.WorkspaceRepository,
	ar *repository.AgentRepository,
	eb *events.Bus,
	jwtSecret string,
	jwtExpiry time.Duration,
) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo: wr,
		agentRepo:     ar,
		eventBus:      eb,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiry:     jwtExpiry,
	}
}

// CreateWorkspace creates a new workspace and its owner agent, generates an
// invite code in the form "HUB-XXXX", and returns a JWT token for the owner.
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput, ownerAgentID uuid.UUID) (*CreateWorkspaceResponse, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generating invite code: %w", err)
	}

	now := time.Now().UTC()
	workspace := &models.Workspace{
		ID:           uuid.New(),
		Name:         input.Name,
		Description:  input.Description,
		OwnerAgentID: ownerAgentID,
		InviteCode:   inviteCode,
		Status:       models.WorkspaceStatusActive,
		Settings:     make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.workspaceRepo.Create(ctx, workspace); err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	token, err := s.generateToken(ownerAgentID, workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	s.eventBus.Publish(events.NewEvent(events.EventAgentOnline, workspace.ID.String(), map[string]interface{}{
		"workspace_id": workspace.ID,
		"agent_id":     ownerAgentID,
	}))

	return &CreateWorkspaceResponse{
		Workspace: workspace,
		AgentID:   ownerAgentID,
		Token:     token,
	}, nil
}

// GetWorkspace retrieves a workspace by ID.
func (s *WorkspaceService) GetWorkspace(ctx context.Context, id uuid.UUID) (*models.Workspace, error) {
	workspace, err := s.workspaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	return workspace, nil
}

// UpdateWorkspace applies partial updates to a workspace. Only the workspace
// owner may perform updates.
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, id uuid.UUID, input UpdateWorkspaceInput, agentID uuid.UUID) (*models.Workspace, error) {
	workspace, err := s.workspaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}

	if workspace.OwnerAgentID != agentID {
		return nil, fmt.Errorf("only the workspace owner can update the workspace")
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, fmt.Errorf("workspace name cannot be empty")
		}
		workspace.Name = *input.Name
	}
	if input.Description != nil {
		workspace.Description = *input.Description
	}
	if input.Settings != nil {
		for k, v := range input.Settings {
			workspace.Settings[k] = v
		}
	}
	workspace.UpdatedAt = time.Now().UTC()

	if err := s.workspaceRepo.Update(ctx, workspace); err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}

	return workspace, nil
}

// DeleteWorkspace soft-deletes a workspace by setting its status to deleted.
// Only the workspace owner may delete.
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, id uuid.UUID, agentID uuid.UUID) error {
	workspace, err := s.workspaceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	if workspace.OwnerAgentID != agentID {
		return fmt.Errorf("only the workspace owner can delete the workspace")
	}

	workspace.Status = models.WorkspaceStatusDeleted
	workspace.UpdatedAt = time.Now().UTC()

	if err := s.workspaceRepo.Update(ctx, workspace); err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	return nil
}

// JoinWorkspace allows an agent to join a workspace using an invite code.
// It creates a new agent record and returns a JWT token.
func (s *WorkspaceService) JoinWorkspace(ctx context.Context, input JoinWorkspaceInput) (*JoinWorkspaceResponse, error) {
	if input.InviteCode == "" {
		return nil, fmt.Errorf("invite code is required")
	}
	if input.AgentName == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if !models.IsValidRole(input.AgentRole) {
		return nil, fmt.Errorf("invalid agent role: %s", input.AgentRole)
	}

	workspace, err := s.workspaceRepo.GetByInviteCode(ctx, input.InviteCode)
	if err != nil {
		return nil, fmt.Errorf("invalid invite code: %w", err)
	}

	if workspace.Status != models.WorkspaceStatusActive {
		return nil, fmt.Errorf("workspace is not active")
	}

	capabilities := input.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}

	now := time.Now().UTC()
	agent := &models.Agent{
		ID:            uuid.New(),
		WorkspaceID:   workspace.ID,
		Name:          input.AgentName,
		Role:          input.AgentRole,
		Status:        models.AgentStatusOnline,
		Capabilities:  capabilities,
		LastHeartbeat: now,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.agentRepo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("creating agent: %w", err)
	}

	token, err := s.generateToken(agent.ID, workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	s.eventBus.Publish(events.NewEvent(events.EventAgentOnline, workspace.ID.String(), map[string]interface{}{
		"workspace_id": workspace.ID,
		"agent_id":     agent.ID,
		"agent_name":   agent.Name,
	}))

	return &JoinWorkspaceResponse{
		Workspace: workspace,
		Agent:     agent,
		Token:     token,
	}, nil
}

// LeaveWorkspace removes an agent from a workspace. The owner cannot leave.
func (s *WorkspaceService) LeaveWorkspace(ctx context.Context, workspaceID uuid.UUID, agentID uuid.UUID) error {
	workspace, err := s.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	if workspace.OwnerAgentID == agentID {
		return fmt.Errorf("workspace owner cannot leave; transfer ownership or delete the workspace")
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("getting agent: %w", err)
	}
	if agent.WorkspaceID != workspaceID {
		return fmt.Errorf("agent does not belong to this workspace")
	}

	if err := s.agentRepo.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("removing agent: %w", err)
	}

	s.eventBus.Publish(events.NewEvent(events.EventAgentOffline, workspaceID.String(), map[string]interface{}{
		"workspace_id": workspaceID,
		"agent_id":     agentID,
	}))

	return nil
}

// ListAgents returns all agents in a workspace.
func (s *WorkspaceService) ListAgents(ctx context.Context, workspaceID uuid.UUID) ([]models.Agent, error) {
	agents, err := s.agentRepo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}
	return agents, nil
}

// UpdateAgentStatus updates the online/offline/busy status of an agent.
func (s *WorkspaceService) UpdateAgentStatus(ctx context.Context, agentID uuid.UUID, status string) error {
	if !models.IsValidAgentStatus(status) {
		return fmt.Errorf("invalid agent status: %s", status)
	}

	if err := s.agentRepo.UpdateStatus(ctx, agentID, status); err != nil {
		return fmt.Errorf("updating agent status: %w", err)
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("getting agent after status update: %w", err)
	}

	eventType := events.EventAgentOnline
	if status == models.AgentStatusOffline {
		eventType = events.EventAgentOffline
	}
	s.eventBus.Publish(events.NewEvent(eventType, agent.WorkspaceID.String(), map[string]interface{}{
		"agent_id": agentID,
		"status":   status,
	}))

	return nil
}

// Heartbeat updates the agent's heartbeat timestamp and optionally updates
// status and metadata.
func (s *WorkspaceService) Heartbeat(ctx context.Context, agentID uuid.UUID, input HeartbeatInput) (*models.Agent, error) {
	if err := s.agentRepo.UpdateHeartbeat(ctx, agentID); err != nil {
		return nil, fmt.Errorf("updating heartbeat: %w", err)
	}

	if input.Status != "" && models.IsValidAgentStatus(input.Status) {
		if err := s.agentRepo.UpdateStatus(ctx, agentID, input.Status); err != nil {
			return nil, fmt.Errorf("updating status during heartbeat: %w", err)
		}
	}

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting agent after heartbeat: %w", err)
	}

	return agent, nil
}

// ---------- helpers ----------

// generateInviteCode produces a code in the format "HUB-XXXX" using crypto/rand.
func generateInviteCode() (string, error) {
	code := make([]byte, 4)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			return "", fmt.Errorf("generating random char: %w", err)
		}
		code[i] = inviteCodeChars[n.Int64()]
	}
	return "HUB-" + string(code), nil
}

// generateToken creates a JWT token for the given agent and workspace.
func (s *WorkspaceService) generateToken(agentID, workspaceID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"agent_id":     agentID.String(),
		"workspace_id": workspaceID.String(),
		"iat":          now.Unix(),
		"exp":          now.Add(s.jwtExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}
