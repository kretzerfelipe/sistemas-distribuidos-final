package teams

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/storage"
)

var (
	pngHeader  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0x0D, 'I', 'H', 'D', 'R'}
	jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
)

type storedObject struct {
	contentType string
	data        []byte
}

type fakeStorage struct {
	objects map[string]storedObject
	failing bool
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{objects: map[string]storedObject{}}
}

func (s *fakeStorage) Put(_ context.Context, key, contentType string, data []byte) error {
	if s.failing {
		return storage.ErrUnavailable
	}
	s.objects[key] = storedObject{contentType: contentType, data: data}
	return nil
}

func (s *fakeStorage) Get(_ context.Context, key string) (*storage.Object, error) {
	if s.failing {
		return nil, storage.ErrUnavailable
	}
	object, found := s.objects[key]
	if !found {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.Object{Body: io.NopCloser(bytes.NewReader(object.data)), ContentType: object.contentType, Size: int64(len(object.data))}, nil
}

func (s *fakeStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (r *inMemoryRepository) UpdatePhoto(_ context.Context, id uuid.UUID, key, contentType string, updatedAt time.Time) error {
	team, found := r.teamsByID[id]
	if !found {
		return entities.ErrNotFound
	}
	team.PhotoKey, team.PhotoContentType, team.PhotoUpdatedAt = key, contentType, &updatedAt
	r.teamsByID[id] = team
	return nil
}

func (r *inMemoryRepository) ClearPhoto(_ context.Context, id uuid.UUID) error {
	team, found := r.teamsByID[id]
	if !found {
		return entities.ErrNotFound
	}
	team.PhotoKey, team.PhotoContentType, team.PhotoUpdatedAt = "", "", nil
	r.teamsByID[id] = team
	return nil
}

func TestTheOwnerSetsThePhotoAndAnyoneCanReadIt(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")

	updated, err := env.useCase.SetPhoto(context.Background(), owner, team.ID, pngHeader)
	photo, readErr := env.useCase.GetPhoto(context.Background(), team.ID)

	if err != nil || updated.PhotoUpdatedAt == nil || updated.PhotoContentType != "image/png" {
		t.Fatalf("the photo must be saved as a PNG, got %+v (err=%v)", updated, err)
	}
	if readErr != nil || photo.ContentType != "image/png" {
		t.Fatalf("the photo must be readable, got %+v (err=%v)", photo, readErr)
	}
	stored, _ := io.ReadAll(photo.Body)
	if !bytes.Equal(stored, pngHeader) {
		t.Error("the bytes read must be the bytes sent")
	}
}

func TestReplacingThePhotoDeletesTheOldObject(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	first, _ := env.useCase.SetPhoto(context.Background(), owner, team.ID, pngHeader)

	second, err := env.useCase.SetPhoto(context.Background(), owner, team.ID, jpegHeader)

	if err != nil || second.PhotoKey == first.PhotoKey || second.PhotoContentType != "image/jpeg" {
		t.Fatalf("the new photo must have its own key and type, got %+v (err=%v)", second, err)
	}
	_, oldStillExists := env.storage.objects[first.PhotoKey]
	if oldStillExists || len(env.storage.objects) != 1 {
		t.Errorf("only the new object must remain, got %d objects", len(env.storage.objects))
	}
}

func TestOnlyTheOwnerOrAnAdminCanChangeThePhoto(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")

	byStranger, strangerErr := env.useCase.SetPhoto(context.Background(), newTeamCaller(), team.ID, pngHeader)
	byAdmin, adminErr := env.useCase.SetPhoto(context.Background(), newAdminCaller(), team.ID, pngHeader)

	if !errors.Is(strangerErr, entities.ErrForbidden) || byStranger != nil || len(env.storage.objects) != 1 {
		t.Errorf("a stranger must be forbidden and store nothing, got %v / %d objects", strangerErr, len(env.storage.objects))
	}
	if adminErr != nil || byAdmin == nil {
		t.Errorf("an admin must be allowed: %v", adminErr)
	}
}

func TestInvalidPhotosAreRejectedBeforeReachingTheStorage(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	scenarios := map[string]struct {
		data []byte
		want error
	}{
		"empty":        {nil, entities.ErrInvalidInput},
		"not an image": {[]byte("just some text, not an image at all"), entities.ErrUnsupportedPhoto},
		"too large":    {append(append([]byte{}, pngHeader...), make([]byte, MaxPhotoBytes)...), entities.ErrPhotoTooLarge},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			_, err := env.useCase.SetPhoto(context.Background(), owner, team.ID, scenario.data)

			if !errors.Is(err, scenario.want) {
				t.Errorf("want %v, got %v", scenario.want, err)
			}
		})
	}
	if len(env.storage.objects) != 0 {
		t.Errorf("nothing may be stored, got %d objects", len(env.storage.objects))
	}
}

func TestAStorageOutageIsReportedAsUnavailableAndKeepsTheTeamUnchanged(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	env.storage.failing = true

	_, setErr := env.useCase.SetPhoto(context.Background(), owner, team.ID, pngHeader)

	reloaded, _ := env.useCase.GetByID(context.Background(), team.ID)
	if !errors.Is(setErr, entities.ErrStorageUnavailable) || reloaded.PhotoUpdatedAt != nil {
		t.Errorf("want ErrStorageUnavailable and no photo, got %v / %+v", setErr, reloaded)
	}
}

func TestReadingThePhotoOfATeamWithoutOneIsNotFound(t *testing.T) {
	env := newTestEnvironment()
	team := env.createTeam(t, newTeamCaller(), "Furb FC")

	photo, err := env.useCase.GetPhoto(context.Background(), team.ID)
	_, unknownErr := env.useCase.GetPhoto(context.Background(), uuid.New())

	if !errors.Is(err, entities.ErrNotFound) || photo != nil || !errors.Is(unknownErr, entities.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v / %v", err, unknownErr)
	}
}

func TestRemovingThePhotoClearsTheTeamAndTheObject(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, _ = env.useCase.SetPhoto(context.Background(), owner, team.ID, pngHeader)

	cleared, err := env.useCase.RemovePhoto(context.Background(), owner, team.ID)

	if err != nil || cleared.PhotoUpdatedAt != nil || len(env.storage.objects) != 0 {
		t.Errorf("the photo must be gone, got %+v / %d objects (err=%v)", cleared, len(env.storage.objects), err)
	}
}

func TestDeletingTheTeamDeletesItsPhoto(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, _ = env.useCase.SetPhoto(context.Background(), owner, team.ID, pngHeader)

	err := env.useCase.Delete(context.Background(), owner, team.ID)

	if err != nil || len(env.storage.objects) != 0 {
		t.Errorf("the photo object must be deleted with the team, got %d objects (err=%v)", len(env.storage.objects), err)
	}
}
