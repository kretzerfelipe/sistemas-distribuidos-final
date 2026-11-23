package teams

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/storage"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo    Repository
	players PlayerService
	storage storage.Client
}

func newUseCase(repo Repository, players PlayerService, photos storage.Client) UseCase {
	return &useCase{repo: repo, players: players, storage: photos}
}

func (uc *useCase) SetPhoto(ctx context.Context, caller *entities.Caller, id uuid.UUID, data []byte) (*entities.Team, error) {
	team, err := uc.findTeamManagedBy(ctx, caller, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to set the photo", err)
	}

	contentType, err := validatePhoto(data)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating team photo", err)
	}

	newKey := fmt.Sprintf("teams/%s/%s", team.ID, uuid.NewString())
	err = uc.storage.Put(ctx, newKey, contentType, data)
	if err != nil {
		return nil, storageFailure("use case error storing team photo", err)
	}

	updatedAt := time.Now().UTC()
	err = uc.repo.UpdatePhoto(ctx, team.ID, newKey, contentType, updatedAt)
	if err != nil {
		uc.discard(ctx, newKey)
		return nil, util.SendMessageWithError("use case error saving team photo", err)
	}

	uc.discard(ctx, team.PhotoKey)
	team.PhotoKey, team.PhotoContentType, team.PhotoUpdatedAt = newKey, contentType, &updatedAt
	return team, nil
}

func (uc *useCase) RemovePhoto(ctx context.Context, caller *entities.Caller, id uuid.UUID) (*entities.Team, error) {
	team, err := uc.findTeamManagedBy(ctx, caller, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to remove the photo", err)
	}

	err = uc.repo.ClearPhoto(ctx, team.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error clearing team photo", err)
	}

	uc.discard(ctx, team.PhotoKey)
	team.PhotoKey, team.PhotoContentType, team.PhotoUpdatedAt = "", "", nil
	return team, nil
}

func (uc *useCase) GetPhoto(ctx context.Context, id uuid.UUID) (*storage.Object, error) {
	team, err := uc.repo.FindTeamByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to read the photo", err)
	}

	teamHasNoPhoto := team.PhotoKey == ""
	if teamHasNoPhoto {
		return nil, fmt.Errorf("%w: team photo", entities.ErrNotFound)
	}

	object, err := uc.storage.Get(ctx, team.PhotoKey)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, fmt.Errorf("%w: team photo", entities.ErrNotFound)
	}
	if err != nil {
		return nil, storageFailure("use case error reading team photo", err)
	}
	return object, nil
}

func (uc *useCase) discard(ctx context.Context, key string) {
	nothingToDiscard := key == ""
	if nothingToDiscard {
		return
	}
	err := uc.storage.Delete(ctx, key)
	if err != nil {
		slog.Warn("could not delete an old team photo", "key", key, "error", err)
	}
}

func storageFailure(message string, err error) error {
	slog.Error(message, "error", err)
	return entities.ErrStorageUnavailable
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

	uc.discard(ctx, team.PhotoKey)
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
