package matches

import (
	"fmt"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

const (
	maxEventMinute = 130
	maxWindowDays  = 31
)

func validateWindow(in entities.ListMatchesInput) error {
	startIsMissing := in.From.IsZero()
	daysAreOutOfRange := in.Days < 1 || in.Days > maxWindowDays

	switch {
	case startIsMissing:
		return fmt.Errorf("%w: from is required", entities.ErrInvalidInput)
	case daysAreOutOfRange:
		return fmt.Errorf("%w: days must be between 1 and %d", entities.ErrInvalidInput, maxWindowDays)
	}
	return nil
}

func validateNewMatch(in entities.CreateMatchInput) error {
	teamsAreTheSame := in.HomeTeamID == in.AwayTeamID
	scheduleIsMissing := in.ScheduledAt.IsZero()

	switch {
	case teamsAreTheSame:
		return fmt.Errorf("%w: a team cannot play against itself", entities.ErrInvalidInput)
	case scheduleIsMissing:
		return fmt.Errorf("%w: scheduledAt is required", entities.ErrInvalidInput)
	}
	return nil
}

func validateSchedule(in entities.RescheduleMatchInput) error {
	scheduleIsMissing := in.ScheduledAt.IsZero()
	if scheduleIsMissing {
		return fmt.Errorf("%w: scheduledAt is required", entities.ErrInvalidInput)
	}
	return nil
}

func eventTypeIsValid(eventType entities.MatchEventType) bool {
	switch eventType {
	case entities.EventGoal, entities.EventOwnGoal, entities.EventShot, entities.EventSave, entities.EventFoul,
		entities.EventYellowCard, entities.EventRedCard, entities.EventSubstitution, entities.EventCorner, entities.EventOffside:
		return true
	}
	return false
}

func validateEvent(in entities.RegisterMatchEventInput, match *entities.Match) error {
	typeIsInvalid := !eventTypeIsValid(in.Type)
	minuteIsOutOfRange := in.Minute < 0 || in.Minute > maxEventMinute
	teamIsNotInTheMatch := in.TeamID != match.HomeTeamID && in.TeamID != match.AwayTeamID

	isSubstitution := in.Type == entities.EventSubstitution
	substitutionNeedsTwoPlayers := isSubstitution && (in.PlayerID == nil || in.SecondaryPlayerID == nil)
	substitutionUsesTheSamePlayer := isSubstitution && in.PlayerID != nil && in.SecondaryPlayerID != nil && *in.PlayerID == *in.SecondaryPlayerID
	onlySubstitutionsHaveASecondaryPlayer := !isSubstitution && in.SecondaryPlayerID != nil

	switch {
	case typeIsInvalid:
		return fmt.Errorf("%w: type is not valid", entities.ErrInvalidInput)
	case minuteIsOutOfRange:
		return fmt.Errorf("%w: minute must be between 0 and %d", entities.ErrInvalidInput, maxEventMinute)
	case teamIsNotInTheMatch:
		return fmt.Errorf("%w: teamId must be the home or the away team of the match", entities.ErrInvalidInput)
	case substitutionNeedsTwoPlayers:
		return fmt.Errorf("%w: a substitution needs playerId (who enters) and secondaryPlayerId (who leaves)", entities.ErrInvalidInput)
	case substitutionUsesTheSamePlayer:
		return fmt.Errorf("%w: a substitution needs two different players", entities.ErrInvalidInput)
	case onlySubstitutionsHaveASecondaryPlayer:
		return fmt.Errorf("%w: secondaryPlayerId is only used by substitutions", entities.ErrInvalidInput)
	}
	return nil
}

func scoreDeltas(eventType entities.MatchEventType, teamID uuid.UUID, match *entities.Match) (homeDelta, awayDelta int) {
	teamIsHome := teamID == match.HomeTeamID

	switch eventType {
	case entities.EventGoal:
		if teamIsHome {
			return 1, 0
		}
		return 0, 1
	case entities.EventOwnGoal:
		if teamIsHome {
			return 0, 1
		}
		return 1, 0
	}
	return 0, 0
}
