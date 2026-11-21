package matches

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo          Repository
	championships ChampionshipService
}

func newUseCase(repo Repository, championships ChampionshipService) UseCase {
	return &useCase{repo: repo, championships: championships}
}

func (uc *useCase) Create(ctx context.Context, in entities.CreateMatchInput) (*entities.Match, error) {
	err := validateNewMatch(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new match", err)
	}

	championship, err := uc.championships.GetByID(ctx, in.ChampionshipID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the championship of the match", err)
	}

	championshipIsFinished := championship.Status == entities.ChampionshipFinished
	if championshipIsFinished {
		return nil, fmt.Errorf("%w: the championship is finished", entities.ErrInvalidState)
	}

	err = uc.ensureTeamsAreEnrolled(ctx, championship.ID, in.HomeTeamID, in.AwayTeamID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error checking enrollment of the match teams", err)
	}

	now := time.Now().UTC()
	match := &entities.Match{
		ID:             uuid.New(),
		ChampionshipID: championship.ID,
		HomeTeamID:     in.HomeTeamID,
		AwayTeamID:     in.AwayTeamID,
		ScheduledAt:    in.ScheduledAt.UTC(),
		Status:         entities.MatchScheduled,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = uc.repo.Insert(ctx, match)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting new match", err)
	}
	return match, nil
}

func (uc *useCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	match, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match by id", err)
	}
	return match, nil
}

func (uc *useCase) ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error) {
	championship, err := uc.championships.GetByID(ctx, championshipID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding championship to list matches", err)
	}

	championshipMatches, err := uc.repo.ListByChampionship(ctx, championship.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing championship matches", err)
	}
	return championshipMatches, nil
}

func (uc *useCase) Reschedule(ctx context.Context, id uuid.UUID, in entities.RescheduleMatchInput) (*entities.Match, error) {
	err := validateSchedule(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new schedule", err)
	}

	match, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match to reschedule", err)
	}

	err = uc.repo.Reschedule(ctx, match.ID, in.ScheduledAt.UTC(), time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error rescheduling match", err)
	}

	rescheduled, err := uc.repo.FindByID(ctx, match.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match after rescheduling", err)
	}
	return rescheduled, nil
}

func (uc *useCase) Delete(ctx context.Context, id uuid.UUID) error {
	match, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return util.SendMessageWithError("use case error finding match to delete", err)
	}

	err = uc.repo.Delete(ctx, match.ID)
	if err != nil {
		return util.SendMessageWithError("use case error deleting match", err)
	}
	return nil
}

func (uc *useCase) Start(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	match, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match to start", err)
	}

	championship, err := uc.championships.GetByID(ctx, match.ChampionshipID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the championship of the match to start", err)
	}

	championshipIsNotInProgress := championship.Status != entities.ChampionshipInProgress
	if championshipIsNotInProgress {
		return nil, fmt.Errorf("%w: the championship must be in progress to start a match", entities.ErrInvalidState)
	}

	return uc.changeStatus(ctx, match.ID, entities.MatchScheduled, entities.MatchInProgress, 0, 0)
}

func (uc *useCase) Finish(ctx context.Context, id uuid.UUID, in entities.FinishMatchInput) (*entities.Match, error) {
	err := validateScore(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating final score", err)
	}
	return uc.changeStatus(ctx, id, entities.MatchInProgress, entities.MatchFinished, in.HomeScore, in.AwayScore)
}

func (uc *useCase) ensureTeamsAreEnrolled(ctx context.Context, championshipID, homeTeamID, awayTeamID uuid.UUID) error {
	homeIsEnrolled, err := uc.championships.TeamIsEnrolled(ctx, championshipID, homeTeamID)
	if err != nil {
		return err
	}
	awayIsEnrolled, err := uc.championships.TeamIsEnrolled(ctx, championshipID, awayTeamID)
	if err != nil {
		return err
	}

	bothTeamsAreEnrolled := homeIsEnrolled && awayIsEnrolled
	if !bothTeamsAreEnrolled {
		return entities.ErrTeamNotEnrolled
	}
	return nil
}

func (uc *useCase) changeStatus(ctx context.Context, id uuid.UUID, from, to entities.MatchStatus, homeScore, awayScore int) (*entities.Match, error) {
	err := uc.repo.Transition(ctx, id, from, to, homeScore, awayScore, time.Now().UTC())
	if err != nil {
		return nil, util.SendMessageWithError("use case error changing match status", err)
	}

	match, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match after status change", err)
	}
	return match, nil
}
