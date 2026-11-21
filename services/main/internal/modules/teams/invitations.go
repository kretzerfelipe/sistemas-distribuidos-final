package teams

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

func (uc *useCase) Invite(ctx context.Context, caller *entities.Caller, teamID, playerID uuid.UUID) (*entities.Invitation, error) {
	team, err := uc.findTeamOwnedBy(ctx, caller, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to invite player", err)
	}

	player, err := uc.players.GetByID(ctx, playerID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding player to invite", err)
	}

	playerIsInATeam, err := uc.repo.PlayerIsInTeam(ctx, player.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error checking whether player is in a team", err)
	}
	if playerIsInATeam {
		return nil, entities.ErrPlayerAlreadyInTeam
	}

	now := time.Now().UTC()
	invitation := &entities.Invitation{
		ID:        uuid.New(),
		TeamID:    team.ID,
		TeamName:  team.Name,
		PlayerID:  player.ID,
		Status:    entities.InvitationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = uc.repo.InsertInvitation(ctx, invitation)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting invitation", err)
	}
	return invitation, nil
}

func (uc *useCase) ListTeamInvitations(ctx context.Context, caller *entities.Caller, teamID uuid.UUID) ([]entities.Invitation, error) {
	team, err := uc.findTeamManagedBy(ctx, caller, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to list invitations", err)
	}

	invitations, err := uc.repo.ListInvitationsByTeam(ctx, team.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing team invitations", err)
	}
	return invitations, nil
}

func (uc *useCase) CancelInvitation(ctx context.Context, caller *entities.Caller, teamID, invitationID uuid.UUID) (*entities.Invitation, error) {
	team, err := uc.findTeamManagedBy(ctx, caller, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to cancel invitation", err)
	}

	invitation, err := uc.repo.FindInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation to cancel", err)
	}

	invitationBelongsToAnotherTeam := invitation.TeamID != team.ID
	if invitationBelongsToAnotherTeam {
		return nil, entities.ErrNotFound
	}

	return uc.changeInvitationStatus(ctx, invitation.ID, entities.InvitationCanceled)
}

func (uc *useCase) ListMyInvitations(ctx context.Context, caller *entities.Caller) ([]entities.Invitation, error) {
	player, err := uc.players.GetByUserID(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the caller's player to list invitations", err)
	}

	invitations, err := uc.repo.ListInvitationsByPlayer(ctx, player.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing the caller's invitations", err)
	}
	return invitations, nil
}

func (uc *useCase) AcceptInvitation(ctx context.Context, caller *entities.Caller, invitationID uuid.UUID) (*entities.Invitation, error) {
	invitation, err := uc.findInvitationAddressedTo(ctx, caller, invitationID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation to accept", err)
	}

	err = uc.repo.AcceptInvitation(ctx, invitation, time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error accepting invitation", err)
	}

	accepted, err := uc.repo.FindInvitationByID(ctx, invitation.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation after accepting", err)
	}
	return accepted, nil
}

func (uc *useCase) DeclineInvitation(ctx context.Context, caller *entities.Caller, invitationID uuid.UUID) (*entities.Invitation, error) {
	invitation, err := uc.findInvitationAddressedTo(ctx, caller, invitationID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation to decline", err)
	}
	return uc.changeInvitationStatus(ctx, invitation.ID, entities.InvitationDeclined)
}

func (uc *useCase) findInvitationAddressedTo(ctx context.Context, caller *entities.Caller, invitationID uuid.UUID) (*entities.Invitation, error) {
	player, err := uc.players.GetByUserID(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the caller's player", err)
	}

	invitation, err := uc.repo.FindInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation", err)
	}

	invitationIsForAnotherPlayer := invitation.PlayerID != player.ID
	if invitationIsForAnotherPlayer {
		return nil, entities.ErrForbidden
	}
	return invitation, nil
}

func (uc *useCase) changeInvitationStatus(ctx context.Context, invitationID uuid.UUID, newStatus entities.InvitationStatus) (*entities.Invitation, error) {
	err := uc.repo.UpdateInvitationStatus(ctx, invitationID, entities.InvitationPending, newStatus, time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error changing invitation status", err)
	}

	invitation, err := uc.repo.FindInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding invitation after status change", err)
	}
	return invitation, nil
}
