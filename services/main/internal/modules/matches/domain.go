package matches

import (
	"fmt"

	"futhub/main/internal/entities"
)

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

func validateScore(in entities.FinishMatchInput) error {
	scoreIsNegative := in.HomeScore < 0 || in.AwayScore < 0
	if scoreIsNegative {
		return fmt.Errorf("%w: scores cannot be negative", entities.ErrInvalidInput)
	}
	return nil
}
