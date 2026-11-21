package championships

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo  Repository
	teams TeamService
}

func newUseCase(repo Repository, teams TeamService) UseCase {
	return &useCase{repo: repo, teams: teams}
}

func (uc *useCase) Create(ctx context.Context, in entities.SaveChampionshipInput) (*entities.Championship, error) {
	validatedInput, err := validate(in, defaultSlotsTotal)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new championship", err)
	}

	now := time.Now().UTC()
	championship := &entities.Championship{
		ID:         uuid.New(),
		Name:       validatedInput.Name,
		SlotsTotal: validatedInput.SlotsTotal,
		Status:     entities.ChampionshipOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = uc.repo.Insert(ctx, championship)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting new championship", err)
	}
	return championship, nil
}

func (uc *useCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error) {
	championship, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship by id", err)
	}
	return championship, nil
}

func (uc *useCase) List(ctx context.Context) ([]entities.Championship, error) {
	allChampionships, err := uc.repo.List(ctx)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing championships", err)
	}
	return allChampionships, nil
}

func (uc *useCase) Update(ctx context.Context, id uuid.UUID, in entities.SaveChampionshipInput) (*entities.Championship, error) {
	championship, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship to update", err)
	}

	championshipIsFinished := championship.Status == entities.ChampionshipFinished
	if championshipIsFinished {
		return nil, fmt.Errorf("%w: a finished championship cannot be edited", entities.ErrInvalidState)
	}

	validatedInput, err := validate(in, championship.SlotsTotal)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating championship update", err)
	}

	slotsAreFewerThanEnrolledTeams := validatedInput.SlotsTotal < championship.SlotsTaken
	if slotsAreFewerThanEnrolledTeams {
		return nil, fmt.Errorf("%w: slotsTotal cannot be lower than the %d teams already enrolled",
			entities.ErrInvalidInput, championship.SlotsTaken)
	}

	championship.Name = validatedInput.Name
	championship.SlotsTotal = validatedInput.SlotsTotal
	championship.UpdatedAt = time.Now().UTC()

	err = uc.repo.Update(ctx, championship)
	if err != nil {
		return nil, util.SendMessageWithError("use case error updating championship", err)
	}
	return championship, nil
}

func (uc *useCase) Delete(ctx context.Context, id uuid.UUID) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return util.SendMessageWithError("use case error deleting championship", err)
	}
	return nil
}

func (uc *useCase) Start(ctx context.Context, id uuid.UUID) (*entities.Championship, error) {
	championship, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship to start", err)
	}

	championshipHasTooFewTeams := championship.SlotsTaken < minTeamsToStart
	if championshipHasTooFewTeams {
		return nil, fmt.Errorf("%w: at least %d teams must be enrolled to start", entities.ErrInvalidState, minTeamsToStart)
	}
	return uc.changeStatus(ctx, id, entities.ChampionshipOpen, entities.ChampionshipInProgress)
}

func (uc *useCase) Finish(ctx context.Context, id uuid.UUID) (*entities.Championship, error) {
	return uc.changeStatus(ctx, id, entities.ChampionshipInProgress, entities.ChampionshipFinished)
}

func (uc *useCase) Enroll(ctx context.Context, caller *entities.Caller, championshipID, teamID uuid.UUID) (*entities.Championship, error) {
	team, err := uc.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding team to enroll", err)
	}

	callerCanManage := callerCanManageTeam(caller, team)
	if !callerCanManage {
		return nil, entities.ErrForbidden
	}

	err = uc.repo.Enroll(ctx, championshipID, team.ID, time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error enrolling team", err)
	}

	championship, err := uc.repo.FindByID(ctx, championshipID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship after enrollment", err)
	}
	return championship, nil
}

func (uc *useCase) Unenroll(ctx context.Context, caller *entities.Caller, championshipID, teamID uuid.UUID) error {
	team, err := uc.teams.GetByID(ctx, teamID)
	if err != nil {
		return util.SendMessageWithError("use case error finding team to unenroll", err)
	}

	callerCanManage := callerCanManageTeam(caller, team)
	if !callerCanManage {
		return entities.ErrForbidden
	}

	championship, err := uc.repo.FindByID(ctx, championshipID)
	if err != nil {
		return util.SendMessageWithError("use case error finding championship to unenroll", err)
	}

	championshipIsNotOpen := championship.Status != entities.ChampionshipOpen
	if championshipIsNotOpen {
		return entities.ErrChampionshipNotOpen
	}

	err = uc.repo.Unenroll(ctx, championship.ID, team.ID)
	if err != nil {
		return util.SendMessageWithError("use case error unenrolling team", err)
	}
	return nil
}

func (uc *useCase) ListTeams(ctx context.Context, championshipID uuid.UUID) ([]entities.Team, error) {
	championship, err := uc.repo.FindByID(ctx, championshipID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship to list teams", err)
	}

	teamIDs, err := uc.repo.ListEnrolledTeamIDs(ctx, championship.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing enrolled team ids", err)
	}

	enrolledTeams, err := uc.teams.GetByIDs(ctx, teamIDs)
	if err != nil {
		return nil, util.SendMessageWithError("use case error loading enrolled teams", err)
	}
	return enrolledTeams, nil
}

func (uc *useCase) TeamIsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error) {
	isEnrolled, err := uc.repo.IsEnrolled(ctx, championshipID, teamID)
	if err != nil {
		return false, util.SendMessageWithError("use case error checking enrollment", err)
	}
	return isEnrolled, nil
}

func (uc *useCase) changeStatus(ctx context.Context, id uuid.UUID, from, to entities.ChampionshipStatus) (*entities.Championship, error) {
	err := uc.repo.Transition(ctx, id, from, to, time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error changing championship status", err)
	}

	championship, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship after status change", err)
	}
	return championship, nil
}
