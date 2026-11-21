package teams

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo    Repository
	players PlayerService
}

func newUseCase(repo Repository, players PlayerService) UseCase {
	return &useCase{repo: repo, players: players}
}

func (uc *useCase) Create(ctx context.Context, caller *entities.Caller, in entities.SaveTeamInput) (*entities.Team, error) {
	validatedInput, err := validate(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new team", err)
	}

	now := time.Now().UTC()
	team := &entities.Team{
		ID:        uuid.New(),
		OwnerID:   caller.UserID,
		Name:      validatedInput.Name,
		City:      validatedInput.City,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = uc.repo.InsertTeam(ctx, team)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting new team", err)
	}
	return team, nil
}

func (uc *useCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error) {
	team, err := uc.repo.FindTeamByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team by id", err)
	}
	return team, nil
}

func (uc *useCase) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error) {
	noIDsWereRequested := len(ids) == 0
	if noIDsWereRequested {
		return []entities.Team{}, nil
	}

	foundTeams, err := uc.repo.FindTeamsByIDs(ctx, ids)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding teams by ids", err)
	}
	return foundTeams, nil
}

func (uc *useCase) List(ctx context.Context) ([]entities.Team, error) {
	allTeams, err := uc.repo.ListTeams(ctx)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing teams", err)
	}
	return allTeams, nil
}

func (uc *useCase) ListMine(ctx context.Context, caller *entities.Caller) ([]entities.Team, error) {
	ownedTeams, err := uc.repo.ListTeamsByOwner(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing the caller's teams", err)
	}
	return ownedTeams, nil
}

func (uc *useCase) Update(ctx context.Context, caller *entities.Caller, id uuid.UUID, in entities.SaveTeamInput) (*entities.Team, error) {
	team, err := uc.findTeamManagedBy(ctx, caller, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to update", err)
	}

	validatedInput, err := validate(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating team update", err)
	}

	team.Name = validatedInput.Name
	team.City = validatedInput.City
	team.UpdatedAt = time.Now().UTC()

	err = uc.repo.UpdateTeam(ctx, team)
	if err != nil {
		return nil, util.SendMessageWithError("use case error updating team", err)
	}
	return team, nil
}

func (uc *useCase) Delete(ctx context.Context, caller *entities.Caller, id uuid.UUID) error {
	team, err := uc.findTeamManagedBy(ctx, caller, id)
	if err != nil {
		return util.SendMessageWithError("use case error finding team to delete", err)
	}

	err = uc.repo.DeleteTeam(ctx, team.ID)
	if err != nil {
		return util.SendMessageWithError("use case error deleting team", err)
	}
	return nil
}

func (uc *useCase) ListPlayers(ctx context.Context, teamID uuid.UUID) ([]entities.Player, error) {
	team, err := uc.repo.FindTeamByID(ctx, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to list players", err)
	}

	playerIDs, err := uc.repo.ListMemberPlayerIDs(ctx, team.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing team member ids", err)
	}

	teamHasNoPlayers := len(playerIDs) == 0
	if teamHasNoPlayers {
		return []entities.Player{}, nil
	}

	roster, err := uc.players.GetByIDs(ctx, playerIDs)
	if err != nil {
		return nil, util.SendMessageWithError("use case error loading team players", err)
	}
	return roster, nil
}

func (uc *useCase) PlayerBelongsToTeam(ctx context.Context, teamID, playerID uuid.UUID) (bool, error) {
	belongsToTeam, err := uc.repo.PlayerIsMemberOfTeam(ctx, teamID, playerID)
	if err != nil {
		return false, util.SendMessageWithError("use case error checking team membership", err)
	}
	return belongsToTeam, nil
}

func (uc *useCase) RemovePlayer(ctx context.Context, caller *entities.Caller, teamID, playerID uuid.UUID) error {
	team, err := uc.findTeamManagedBy(ctx, caller, teamID)
	if err != nil {
		return util.SendMessageWithError("use case error finding team to remove player", err)
	}

	err = uc.repo.DeleteMembership(ctx, team.ID, playerID)
	if err != nil {
		return util.SendMessageWithError("use case error removing player from team", err)
	}
	return nil
}

func (uc *useCase) LeaveTeam(ctx context.Context, caller *entities.Caller) error {
	player, err := uc.players.GetByUserID(ctx, caller.UserID)
	if err != nil {
		return util.SendMessageWithError("use case error finding the caller's player to leave team", err)
	}

	err = uc.repo.DeleteMembershipByPlayer(ctx, player.ID)
	if err != nil {
		return util.SendMessageWithError("use case error leaving team", err)
	}
	return nil
}

func (uc *useCase) GetMyTeam(ctx context.Context, caller *entities.Caller) (*entities.Team, error) {
	player, err := uc.players.GetByUserID(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the caller's player to get the team", err)
	}

	team, err := uc.repo.FindTeamByPlayer(ctx, player.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the team of the player", err)
	}
	return team, nil
}

func (uc *useCase) findTeamManagedBy(ctx context.Context, caller *entities.Caller, teamID uuid.UUID) (*entities.Team, error) {
	team, err := uc.repo.FindTeamByID(ctx, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team", err)
	}

	callerCanManage := callerCanManageTeam(caller, team)
	if !callerCanManage {
		return nil, entities.ErrForbidden
	}
	return team, nil
}

func (uc *useCase) findTeamOwnedBy(ctx context.Context, caller *entities.Caller, teamID uuid.UUID) (*entities.Team, error) {
	team, err := uc.repo.FindTeamByID(ctx, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team", err)
	}

	callerIsOwner := callerOwnsTeam(caller, team)
	if !callerIsOwner {
		return nil, entities.ErrForbidden
	}
	return team, nil
}
