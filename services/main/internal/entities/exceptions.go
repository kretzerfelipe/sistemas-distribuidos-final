package entities

import "futhub/main/internal/platform/util"

var (
	ErrInvalidInput       = util.NewBusinessError("invalid input")
	ErrInvalidCredentials = util.NewBusinessError("invalid email or password")
	ErrUnauthenticated    = util.NewBusinessError("authentication required")
	ErrForbidden          = util.NewBusinessError("you do not have permission for this resource")
	ErrNotFound           = util.NewBusinessError("not found")
	ErrInvalidState       = util.NewBusinessError("operation not allowed in the current state")

	ErrEmailTaken = util.NewBusinessError("email already registered")

	ErrPlayerAlreadyExists = util.NewBusinessError("player profile already exists")

	ErrTeamNameTaken            = util.NewBusinessError("team name already in use")
	ErrTeamInUse                = util.NewBusinessError("team is enrolled in a championship or has matches")
	ErrPlayerAlreadyInTeam      = util.NewBusinessError("player already belongs to a team")
	ErrPlayerNotInTeam          = util.NewBusinessError("player is not in this team")
	ErrInvitationAlreadyPending = util.NewBusinessError("player already has a pending invitation from this team")
	ErrInvitationNotPending     = util.NewBusinessError("invitation is not pending")

	ErrChampionshipFull       = util.NewBusinessError("championship has no free slots")
	ErrChampionshipNotOpen    = util.NewBusinessError("championship is not open for enrollment")
	ErrChampionshipHasMatches = util.NewBusinessError("championship has matches")
	ErrTeamAlreadyEnrolled    = util.NewBusinessError("team is already enrolled in this championship")
	ErrTeamNotEnrolled        = util.NewBusinessError("team is not enrolled in this championship")
)
