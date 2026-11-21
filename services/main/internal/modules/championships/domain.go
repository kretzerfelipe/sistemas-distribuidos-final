package championships

import (
	"fmt"
	"strings"

	"futhub/main/internal/entities"
)

const (
	defaultSlotsTotal = 10
	maxSlotsTotal     = 64
	minTeamsToStart   = 2
	maxNameLength     = 120
)

func validate(in entities.SaveChampionshipInput, defaultSlots int) (*entities.SaveChampionshipInput, error) {
	in.Name = strings.TrimSpace(in.Name)

	slotsWereNotInformed := in.SlotsTotal == 0
	if slotsWereNotInformed {
		in.SlotsTotal = defaultSlots
	}

	nameIsInvalid := in.Name == "" || len(in.Name) > maxNameLength
	slotsAreOutOfRange := in.SlotsTotal < 1 || in.SlotsTotal > maxSlotsTotal

	switch {
	case nameIsInvalid:
		return nil, fmt.Errorf("%w: name is required and must have at most %d characters", entities.ErrInvalidInput, maxNameLength)
	case slotsAreOutOfRange:
		return nil, fmt.Errorf("%w: slotsTotal must be between 1 and %d", entities.ErrInvalidInput, maxSlotsTotal)
	}
	return &in, nil
}

func callerCanManageTeam(caller *entities.Caller, team *entities.Team) bool {
	callerIsAdmin := caller.Role == entities.RoleAdmin
	callerIsOwner := caller.UserID == team.OwnerID
	return callerIsAdmin || callerIsOwner
}
