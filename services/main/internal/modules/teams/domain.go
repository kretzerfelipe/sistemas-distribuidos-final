package teams

import (
	"fmt"
	"strings"

	"futhub/main/internal/entities"
)

const (
	maxTeamNameLength = 100
	maxCityLength     = 100
)

func validate(in entities.SaveTeamInput) (*entities.SaveTeamInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.City = strings.TrimSpace(in.City)

	nameIsInvalid := in.Name == "" || len(in.Name) > maxTeamNameLength
	cityIsTooLong := len(in.City) > maxCityLength

	switch {
	case nameIsInvalid:
		return nil, fmt.Errorf("%w: name is required and must have at most %d characters", entities.ErrInvalidInput, maxTeamNameLength)
	case cityIsTooLong:
		return nil, fmt.Errorf("%w: city must have at most %d characters", entities.ErrInvalidInput, maxCityLength)
	}
	return &in, nil
}

func callerOwnsTeam(caller *entities.Caller, team *entities.Team) bool {
	return caller.UserID == team.OwnerID
}

func callerCanManageTeam(caller *entities.Caller, team *entities.Team) bool {
	callerIsAdmin := caller.Role == entities.RoleAdmin
	return callerIsAdmin || callerOwnsTeam(caller, team)
}
