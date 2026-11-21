package matches

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo          Repository
	championships ChampionshipService
	teams         TeamService
	players       PlayerService
	transactions  Transactions
	outbox        Outbox
}

type matchTeams struct {
	home entities.NamedReference
	away entities.NamedReference
}

func newUseCase(repo Repository, championships ChampionshipService, teams TeamService, players PlayerService, transactions Transactions, outbox Outbox) UseCase {
	return &useCase{
		repo:          repo,
		championships: championships,
		teams:         teams,
		players:       players,
		transactions:  transactions,
		outbox:        outbox,
	}
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

func (uc *useCase) ListWindow(ctx context.Context, in entities.ListMatchesInput) (*entities.MatchWindow, error) {
	err := validateWindow(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating match window", err)
	}

	windowEnd := in.From.AddDate(0, 0, in.Days)
	windowMatches, err := uc.repo.ListScheduledBetween(ctx, in.From, windowEnd)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing matches of the window", err)
	}

	hasMore, err := uc.repo.ExistsScheduledFrom(ctx, windowEnd)
	if err != nil {
		return nil, util.SendMessageWithError("use case error checking for later matches", err)
	}
	return &entities.MatchWindow{From: in.From, To: windowEnd, Matches: windowMatches, HasMore: hasMore}, nil
}

func (uc *useCase) ListLive(ctx context.Context) ([]entities.Match, error) {
	liveMatches, err := uc.repo.ListByStatus(ctx, entities.MatchInProgress)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing live matches", err)
	}
	return liveMatches, nil
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

	teams, err := uc.findMatchTeams(ctx, match)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the teams of the match to start", err)
	}

	startedAt := time.Now().UTC()
	err = uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		locked, err := uc.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}

		matchIsNotScheduled := locked.Status != entities.MatchScheduled
		if matchIsNotScheduled {
			return fmt.Errorf("%w: match must be %s", entities.ErrInvalidState, entities.MatchScheduled)
		}

		err = uc.repo.Transition(ctx, id, entities.MatchScheduled, entities.MatchInProgress, startedAt)
		if err != nil {
			return err
		}

		progress, err := uc.repo.AdvanceSequence(ctx, id, 0, 0, startedAt)
		if err != nil {
			return err
		}

		return uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessageMatchStarted,
			RoutingKey: entities.RoutingKeyMatchStarted,
			Payload: entities.MatchStartedPayload{
				MatchID:      id,
				Sequence:     progress.Sequence,
				Championship: entities.NamedReference{ID: championship.ID, Name: championship.Name},
				HomeTeam:     teams.home,
				AwayTeam:     teams.away,
				ScheduledAt:  match.ScheduledAt,
				StartedAt:    startedAt,
				Score:        entities.MatchScore{Home: progress.HomeScore, Away: progress.AwayScore},
			},
		})
	})
	if err != nil {
		return nil, util.SendMessageWithError("use case error starting match", err)
	}
	return uc.GetByID(ctx, id)
}

func (uc *useCase) Finish(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	finishedAt := time.Now().UTC()

	err := uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		locked, err := uc.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}

		matchIsNotInProgress := locked.Status != entities.MatchInProgress
		if matchIsNotInProgress {
			return fmt.Errorf("%w: match must be %s", entities.ErrInvalidState, entities.MatchInProgress)
		}

		err = uc.repo.Transition(ctx, id, entities.MatchInProgress, entities.MatchFinished, finishedAt)
		if err != nil {
			return err
		}

		progress, err := uc.repo.AdvanceSequence(ctx, id, 0, 0, finishedAt)
		if err != nil {
			return err
		}

		return uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessageMatchFinished,
			RoutingKey: entities.RoutingKeyMatchFinished,
			Payload: entities.MatchFinishedPayload{
				MatchID:    id,
				Sequence:   progress.Sequence,
				FinishedAt: finishedAt,
				Score:      entities.MatchScore{Home: progress.HomeScore, Away: progress.AwayScore},
			},
		})
	})
	if err != nil {
		return nil, util.SendMessageWithError("use case error finishing match", err)
	}
	return uc.GetByID(ctx, id)
}

func (uc *useCase) RegisterEvent(ctx context.Context, matchID uuid.UUID, in entities.RegisterMatchEventInput) (*entities.MatchEvent, error) {
	match, err := uc.repo.FindByID(ctx, matchID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match to register an event", err)
	}

	err = validateEvent(in, match)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating match event", err)
	}

	playerName, err := uc.findPlayerName(ctx, in.TeamID, in.PlayerID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the player of the event", err)
	}
	secondaryPlayerName, err := uc.findPlayerName(ctx, in.TeamID, in.SecondaryPlayerID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the secondary player of the event", err)
	}

	var registered *entities.MatchEvent
	registeredAt := time.Now().UTC()

	err = uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		locked, err := uc.repo.FindByIDForUpdate(ctx, matchID)
		if err != nil {
			return err
		}

		matchIsNotInProgress := locked.Status != entities.MatchInProgress
		if matchIsNotInProgress {
			return fmt.Errorf("%w: events can only be registered while the match is in progress", entities.ErrInvalidState)
		}

		err = uc.ensurePlayersWereNotSentOff(ctx, matchID, in.PlayerID, in.SecondaryPlayerID)
		if err != nil {
			return err
		}

		homeDelta, awayDelta := scoreDeltas(in.Type, in.TeamID, locked)
		progress, err := uc.repo.AdvanceSequence(ctx, matchID, homeDelta, awayDelta, registeredAt)
		if err != nil {
			return err
		}

		event := &entities.MatchEvent{
			ID:                  uuid.New(),
			MatchID:             matchID,
			Sequence:            progress.Sequence,
			Type:                in.Type,
			Minute:              in.Minute,
			TeamID:              in.TeamID,
			PlayerID:            in.PlayerID,
			PlayerName:          playerName,
			SecondaryPlayerID:   in.SecondaryPlayerID,
			SecondaryPlayerName: secondaryPlayerName,
			CreatedAt:           registeredAt,
		}
		err = uc.repo.InsertEvent(ctx, event)
		if err != nil {
			return err
		}

		err = uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessageMatchEventRegistered,
			RoutingKey: entities.RoutingKeyMatchEventRegistered,
			Payload: entities.MatchEventRegisteredPayload{
				MatchID:  matchID,
				Sequence: progress.Sequence,
				Event:    NewMatchEventResponse(event),
				Score:    entities.MatchScore{Home: progress.HomeScore, Away: progress.AwayScore},
			},
		})
		if err != nil {
			return err
		}

		registered = event
		return nil
	})
	if err != nil {
		return nil, util.SendMessageWithError("use case error registering match event", err)
	}
	return registered, nil
}

func (uc *useCase) RemoveEvent(ctx context.Context, matchID, eventID uuid.UUID) error {
	removedAt := time.Now().UTC()

	err := uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		locked, err := uc.repo.FindByIDForUpdate(ctx, matchID)
		if err != nil {
			return err
		}

		matchIsNotInProgress := locked.Status != entities.MatchInProgress
		if matchIsNotInProgress {
			return fmt.Errorf("%w: events can only be removed while the match is in progress", entities.ErrInvalidState)
		}

		event, err := uc.repo.FindEvent(ctx, matchID, eventID)
		if err != nil {
			return err
		}

		err = uc.repo.DeleteEvent(ctx, matchID, eventID)
		if err != nil {
			return err
		}

		homeDelta, awayDelta := scoreDeltas(event.Type, event.TeamID, locked)
		progress, err := uc.repo.AdvanceSequence(ctx, matchID, -homeDelta, -awayDelta, removedAt)
		if err != nil {
			return err
		}

		return uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessageMatchEventRemoved,
			RoutingKey: entities.RoutingKeyMatchEventRemoved,
			Payload: entities.MatchEventRemovedPayload{
				MatchID:  matchID,
				Sequence: progress.Sequence,
				EventID:  eventID,
				Score:    entities.MatchScore{Home: progress.HomeScore, Away: progress.AwayScore},
			},
		})
	})
	if err != nil {
		return util.SendMessageWithError("use case error removing match event", err)
	}
	return nil
}

func (uc *useCase) ListEvents(ctx context.Context, matchID uuid.UUID) ([]entities.MatchEvent, error) {
	match, err := uc.repo.FindByID(ctx, matchID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding match to list events", err)
	}

	matchEvents, err := uc.repo.ListEvents(ctx, match.ID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing match events", err)
	}
	return matchEvents, nil
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

func (uc *useCase) findMatchTeams(ctx context.Context, match *entities.Match) (*matchTeams, error) {
	found, err := uc.teams.GetByIDs(ctx, []uuid.UUID{match.HomeTeamID, match.AwayTeamID})
	if err != nil {
		return nil, err
	}

	var teams matchTeams
	homeWasFound, awayWasFound := false, false
	for _, team := range found {
		isHome := team.ID == match.HomeTeamID
		isAway := team.ID == match.AwayTeamID
		if isHome {
			teams.home = entities.NamedReference{ID: team.ID, Name: team.Name}
			homeWasFound = true
		}
		if isAway {
			teams.away = entities.NamedReference{ID: team.ID, Name: team.Name}
			awayWasFound = true
		}
	}

	bothTeamsWereFound := homeWasFound && awayWasFound
	if !bothTeamsWereFound {
		return nil, fmt.Errorf("%w: team of the match", entities.ErrNotFound)
	}
	return &teams, nil
}

func (uc *useCase) findPlayerName(ctx context.Context, teamID uuid.UUID, playerID *uuid.UUID) (string, error) {
	playerWasNotInformed := playerID == nil
	if playerWasNotInformed {
		return "", nil
	}

	player, err := uc.players.GetByID(ctx, *playerID)
	if err != nil {
		return "", err
	}

	playerBelongsToTheTeam, err := uc.teams.PlayerBelongsToTeam(ctx, teamID, player.ID)
	if err != nil {
		return "", err
	}
	if !playerBelongsToTheTeam {
		return "", fmt.Errorf("%w: the player does not belong to the team", entities.ErrInvalidInput)
	}
	return player.Name, nil
}

func (uc *useCase) ensurePlayersWereNotSentOff(ctx context.Context, matchID uuid.UUID, playerIDs ...*uuid.UUID) error {
	for _, playerID := range playerIDs {
		playerWasNotInformed := playerID == nil
		if playerWasNotInformed {
			continue
		}

		wasSentOff, err := uc.repo.PlayerHasRedCard(ctx, matchID, *playerID)
		if err != nil {
			return err
		}
		if wasSentOff {
			return fmt.Errorf("%w: the player received a red card and has no more plays in this match", entities.ErrInvalidState)
		}
	}
	return nil
}
