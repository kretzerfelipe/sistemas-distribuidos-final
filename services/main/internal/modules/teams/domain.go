package teams

import (
	"fmt"
	"net/http"
	"strings"

	"futhub/main/internal/entities"
)

const (
	maxTeamNameLength = 100
	maxCityLength     = 100
	MaxPhotoBytes     = 2 << 20
	sniffedBytes      = 512
)

var allowedPhotoTypes = map[string]struct{}{"image/png": {}, "image/jpeg": {}, "image/webp": {}}

func validatePhoto(data []byte) (string, error) {
	photoIsEmpty := len(data) == 0
	photoIsTooLarge := len(data) > MaxPhotoBytes
	switch {
	case photoIsEmpty:
		return "", fmt.Errorf("%w: the photo is empty", entities.ErrInvalidInput)
	case photoIsTooLarge:
		return "", entities.ErrPhotoTooLarge
	}

	head := data
	if len(head) > sniffedBytes {
		head = head[:sniffedBytes]
	}
	contentType := http.DetectContentType(head)
	_, isAllowed := allowedPhotoTypes[contentType]
	if !isAllowed {
		return "", entities.ErrUnsupportedPhoto
	}
	return contentType, nil
}

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
