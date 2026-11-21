package players

import (
	"fmt"
	"strings"

	"futhub/main/internal/entities"
)

const (
	maxNameLength  = 120
	minShirtNumber = 1
	maxShirtNumber = 99
)

func positionIsValid(position entities.Position) bool {
	switch position {
	case entities.PositionGoalkeeper, entities.PositionDefender, entities.PositionMidfielder, entities.PositionForward:
		return true
	}
	return false
}

func validate(in entities.SavePlayerInput) (*entities.SavePlayerInput, error) {
	in.Name = strings.TrimSpace(in.Name)

	nameIsInvalid := in.Name == "" || len(in.Name) > maxNameLength
	positionIsInvalid := !positionIsValid(in.Position)
	shirtNumberIsInvalid := in.ShirtNumber < minShirtNumber || in.ShirtNumber > maxShirtNumber

	switch {
	case nameIsInvalid:
		return nil, fmt.Errorf("%w: name is required and must have at most %d characters", entities.ErrInvalidInput, maxNameLength)
	case positionIsInvalid:
		return nil, fmt.Errorf("%w: position must be GOALKEEPER, DEFENDER, MIDFIELDER or FORWARD", entities.ErrInvalidInput)
	case shirtNumberIsInvalid:
		return nil, fmt.Errorf("%w: shirtNumber must be between %d and %d", entities.ErrInvalidInput, minShirtNumber, maxShirtNumber)
	}
	return &in, nil
}
