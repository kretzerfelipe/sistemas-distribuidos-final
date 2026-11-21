package teams

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func NewTeamResponse(team *entities.Team) entities.TeamResponse {
	return entities.TeamResponse{ID: team.ID, OwnerID: team.OwnerID, Name: team.Name, City: team.City, CreatedAt: team.CreatedAt}
}

func NewInvitationResponse(invitation *entities.Invitation) entities.InvitationResponse {
	return entities.InvitationResponse{
		ID:        invitation.ID,
		TeamID:    invitation.TeamID,
		TeamName:  invitation.TeamName,
		PlayerID:  invitation.PlayerID,
		Status:    invitation.Status,
		CreatedAt: invitation.CreatedAt,
	}
}

type PlayerService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Player, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Player, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Player, error)
}

type UseCase interface {
	Create(ctx context.Context, caller *entities.Caller, in entities.SaveTeamInput) (*entities.Team, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error)
	List(ctx context.Context) ([]entities.Team, error)
	ListMine(ctx context.Context, caller *entities.Caller) ([]entities.Team, error)
	Update(ctx context.Context, caller *entities.Caller, id uuid.UUID, in entities.SaveTeamInput) (*entities.Team, error)
	Delete(ctx context.Context, caller *entities.Caller, id uuid.UUID) error

	ListPlayers(ctx context.Context, teamID uuid.UUID) ([]entities.Player, error)
	PlayerBelongsToTeam(ctx context.Context, teamID, playerID uuid.UUID) (bool, error)
	RemovePlayer(ctx context.Context, caller *entities.Caller, teamID, playerID uuid.UUID) error
	LeaveTeam(ctx context.Context, caller *entities.Caller) error

	Invite(ctx context.Context, caller *entities.Caller, teamID, playerID uuid.UUID) (*entities.Invitation, error)
	ListTeamInvitations(ctx context.Context, caller *entities.Caller, teamID uuid.UUID) ([]entities.Invitation, error)
	CancelInvitation(ctx context.Context, caller *entities.Caller, teamID, invitationID uuid.UUID) (*entities.Invitation, error)
	ListMyInvitations(ctx context.Context, caller *entities.Caller) ([]entities.Invitation, error)
	AcceptInvitation(ctx context.Context, caller *entities.Caller, invitationID uuid.UUID) (*entities.Invitation, error)
	DeclineInvitation(ctx context.Context, caller *entities.Caller, invitationID uuid.UUID) (*entities.Invitation, error)
}

type Repository interface {
	InsertTeam(ctx context.Context, team *entities.Team) error
	FindTeamByID(ctx context.Context, id uuid.UUID) (*entities.Team, error)
	FindTeamsByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error)
	ListTeams(ctx context.Context) ([]entities.Team, error)
	ListTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]entities.Team, error)
	UpdateTeam(ctx context.Context, team *entities.Team) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error

	ListMemberPlayerIDs(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error)
	PlayerIsInTeam(ctx context.Context, playerID uuid.UUID) (bool, error)
	PlayerIsMemberOfTeam(ctx context.Context, teamID, playerID uuid.UUID) (bool, error)
	DeleteMembership(ctx context.Context, teamID, playerID uuid.UUID) error
	DeleteMembershipByPlayer(ctx context.Context, playerID uuid.UUID) error

	InsertInvitation(ctx context.Context, invitation *entities.Invitation) error
	FindInvitationByID(ctx context.Context, id uuid.UUID) (*entities.Invitation, error)
	ListInvitationsByTeam(ctx context.Context, teamID uuid.UUID) ([]entities.Invitation, error)
	ListInvitationsByPlayer(ctx context.Context, playerID uuid.UUID) ([]entities.Invitation, error)
	UpdateInvitationStatus(ctx context.Context, id uuid.UUID, from, to entities.InvitationStatus, updatedAt time.Time) error
	AcceptInvitation(ctx context.Context, invitation *entities.Invitation, acceptedAt time.Time) error
}
