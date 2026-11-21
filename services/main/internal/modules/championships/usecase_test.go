package championships

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type inMemoryRepository struct {
	championshipsByID map[uuid.UUID]entities.Championship
	enrolledTeamIDs   map[uuid.UUID][]uuid.UUID
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		championshipsByID: map[uuid.UUID]entities.Championship{},
		enrolledTeamIDs:   map[uuid.UUID][]uuid.UUID{},
	}
}

func (r *inMemoryRepository) Insert(_ context.Context, championship *entities.Championship) error {
	r.championshipsByID[championship.ID] = *championship
	return nil
}

func (r *inMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Championship, error) {
	championship, found := r.championshipsByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &championship, nil
}

func (r *inMemoryRepository) List(context.Context) ([]entities.Championship, error) {
	all := []entities.Championship{}
	for _, championship := range r.championshipsByID {
		all = append(all, championship)
	}
	return all, nil
}

func (r *inMemoryRepository) Update(_ context.Context, championship *entities.Championship) error {
	_, exists := r.championshipsByID[championship.ID]
	if !exists {
		return entities.ErrNotFound
	}
	r.championshipsByID[championship.ID] = *championship
	return nil
}

func (r *inMemoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	_, exists := r.championshipsByID[id]
	if !exists {
		return entities.ErrNotFound
	}
	delete(r.championshipsByID, id)
	return nil
}

func (r *inMemoryRepository) Transition(_ context.Context, id uuid.UUID, from, to entities.ChampionshipStatus, _ time.Time) error {
	championship, found := r.championshipsByID[id]
	if !found {
		return entities.ErrNotFound
	}
	championshipIsNotInTheExpectedStatus := championship.Status != from
	if championshipIsNotInTheExpectedStatus {
		return entities.ErrInvalidState
	}
	championship.Status = to
	r.championshipsByID[id] = championship
	return nil
}

func (r *inMemoryRepository) Enroll(_ context.Context, championshipID, teamID uuid.UUID, _ time.Time) error {
	championship, found := r.championshipsByID[championshipID]
	if !found {
		return entities.ErrNotFound
	}
	if championship.Status != entities.ChampionshipOpen {
		return entities.ErrChampionshipNotOpen
	}
	if championship.SlotsTaken >= championship.SlotsTotal {
		return entities.ErrChampionshipFull
	}
	for _, enrolledID := range r.enrolledTeamIDs[championshipID] {
		if enrolledID == teamID {
			return entities.ErrTeamAlreadyEnrolled
		}
	}
	championship.SlotsTaken++
	r.championshipsByID[championshipID] = championship
	r.enrolledTeamIDs[championshipID] = append(r.enrolledTeamIDs[championshipID], teamID)
	return nil
}

func (r *inMemoryRepository) Unenroll(_ context.Context, championshipID, teamID uuid.UUID) error {
	remaining := []uuid.UUID{}
	teamWasEnrolled := false
	for _, enrolledID := range r.enrolledTeamIDs[championshipID] {
		if enrolledID == teamID {
			teamWasEnrolled = true
			continue
		}
		remaining = append(remaining, enrolledID)
	}
	if !teamWasEnrolled {
		return entities.ErrTeamNotEnrolled
	}
	r.enrolledTeamIDs[championshipID] = remaining
	championship := r.championshipsByID[championshipID]
	championship.SlotsTaken--
	r.championshipsByID[championshipID] = championship
	return nil
}

func (r *inMemoryRepository) ListEnrolledTeamIDs(_ context.Context, championshipID uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID{}, r.enrolledTeamIDs[championshipID]...), nil
}

func (r *inMemoryRepository) IsEnrolled(_ context.Context, championshipID, teamID uuid.UUID) (bool, error) {
	for _, enrolledID := range r.enrolledTeamIDs[championshipID] {
		if enrolledID == teamID {
			return true, nil
		}
	}
	return false, nil
}

type fakeTeams struct {
	byID map[uuid.UUID]entities.Team
}

func (f *fakeTeams) addTeamOwnedBy(ownerID uuid.UUID) entities.Team {
	team := entities.Team{ID: uuid.New(), OwnerID: ownerID, Name: "Team " + ownerID.String()[:6]}
	f.byID[team.ID] = team
	return team
}

func (f *fakeTeams) GetByID(_ context.Context, id uuid.UUID) (*entities.Team, error) {
	team, found := f.byID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &team, nil
}

func (f *fakeTeams) GetByIDs(_ context.Context, ids []uuid.UUID) ([]entities.Team, error) {
	found := []entities.Team{}
	for _, id := range ids {
		if team, exists := f.byID[id]; exists {
			found = append(found, team)
		}
	}
	return found, nil
}

type testEnvironment struct {
	useCase UseCase
	repo    *inMemoryRepository
	teams   *fakeTeams
}

func newTestEnvironment() testEnvironment {
	repo := newInMemoryRepository()
	teams := &fakeTeams{byID: map[uuid.UUID]entities.Team{}}
	return testEnvironment{useCase: newUseCase(repo, teams), repo: repo, teams: teams}
}

func newTeamCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleTeam}
}

func newAdminCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}
}

func (e testEnvironment) createChampionship(t *testing.T, slots int) *entities.Championship {
	t.Helper()
	championship, err := e.useCase.Create(context.Background(), entities.SaveChampionshipInput{Name: "Copa Furb", SlotsTotal: slots})
	if err != nil {
		t.Fatal(err)
	}
	return championship
}

func (e testEnvironment) enrollNewTeam(t *testing.T, championship *entities.Championship) entities.Team {
	t.Helper()
	owner := newTeamCaller()
	team := e.teams.addTeamOwnedBy(owner.UserID)
	_, err := e.useCase.Enroll(context.Background(), owner, championship.ID, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	return team
}

func TestCreateUsesTenSlotsByDefaultAndStartsOpen(t *testing.T) {
	env := newTestEnvironment()

	championship := env.createChampionship(t, 0)

	if championship.SlotsTotal != 10 || championship.SlotsTaken != 0 || championship.Status != entities.ChampionshipOpen {
		t.Errorf("unexpected championship: %+v", championship)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	scenarios := map[string]entities.SaveChampionshipInput{
		"blank name":     {Name: "  ", SlotsTotal: 10},
		"name too long":  {Name: strings.Repeat("a", maxNameLength+1), SlotsTotal: 10},
		"negative slots": {Name: "Copa", SlotsTotal: -1},
		"too many slots": {Name: "Copa", SlotsTotal: maxSlotsTotal + 1},
	}

	for name, input := range scenarios {
		t.Run(name, func(t *testing.T) {
			championship, err := newTestEnvironment().useCase.Create(context.Background(), input)

			if !errors.Is(err, entities.ErrInvalidInput) || championship != nil {
				t.Errorf("want ErrInvalidInput and nil championship, got %v / %v", err, championship)
			}
		})
	}
}

func TestUpdateChangesNameAndSlotsAndKeepsSlotsWhenOmitted(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 8)

	renamed, renameErr := env.useCase.Update(context.Background(), championship.ID, entities.SaveChampionshipInput{Name: "Copa Nova"})
	resized, resizeErr := env.useCase.Update(context.Background(), championship.ID, entities.SaveChampionshipInput{Name: "Copa Nova", SlotsTotal: 16})

	if renameErr != nil || renamed.Name != "Copa Nova" || renamed.SlotsTotal != 8 {
		t.Errorf("omitted slots must be kept: %+v / %v", renamed, renameErr)
	}
	if resizeErr != nil || resized.SlotsTotal != 16 {
		t.Errorf("slots must change: %+v / %v", resized, resizeErr)
	}
}

func TestUpdateRejectsFewerSlotsThanEnrolledTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 8)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)

	updated, err := env.useCase.Update(context.Background(), championship.ID, entities.SaveChampionshipInput{Name: "Copa", SlotsTotal: 2})

	if !errors.Is(err, entities.ErrInvalidInput) || updated != nil {
		t.Errorf("want ErrInvalidInput and nil championship, got %v / %v", err, updated)
	}
}

func TestUpdateRejectsAFinishedChampionship(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 8)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)
	env.useCase.Start(context.Background(), championship.ID)
	env.useCase.Finish(context.Background(), championship.ID)

	updated, err := env.useCase.Update(context.Background(), championship.ID, entities.SaveChampionshipInput{Name: "Late edit"})

	if !errors.Is(err, entities.ErrInvalidState) || updated != nil {
		t.Errorf("want ErrInvalidState and nil championship, got %v / %v", err, updated)
	}
}

func TestEnrollTakesASlotForTheOwnerAndForTheAdmin(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	owner := newTeamCaller()
	ownerTeam := env.teams.addTeamOwnedBy(owner.UserID)
	adminTeam := env.teams.addTeamOwnedBy(uuid.New())

	afterOwner, ownerErr := env.useCase.Enroll(context.Background(), owner, championship.ID, ownerTeam.ID)
	afterAdmin, adminErr := env.useCase.Enroll(context.Background(), newAdminCaller(), championship.ID, adminTeam.ID)

	if ownerErr != nil || afterOwner.SlotsTaken != 1 {
		t.Errorf("the owner must enroll: %+v / %v", afterOwner, ownerErr)
	}
	if adminErr != nil || afterAdmin.SlotsTaken != 2 {
		t.Errorf("the admin must enroll any team: %+v / %v", afterAdmin, adminErr)
	}
}

func TestEnrollIsForbiddenForSomeoneWhoDoesNotOwnTheTeam(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	team := env.teams.addTeamOwnedBy(uuid.New())

	updated, err := env.useCase.Enroll(context.Background(), newTeamCaller(), championship.ID, team.ID)

	if !errors.Is(err, entities.ErrForbidden) || updated != nil {
		t.Errorf("want ErrForbidden and nil championship, got %v / %v", err, updated)
	}
	stored, _ := env.useCase.GetByID(context.Background(), championship.ID)
	if stored.SlotsTaken != 0 {
		t.Errorf("no slot may be taken, got %d", stored.SlotsTaken)
	}
}

func TestEnrollFailsForUnknownTeamAndUnknownChampionship(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	owner := newTeamCaller()
	team := env.teams.addTeamOwnedBy(owner.UserID)

	_, unknownTeamErr := env.useCase.Enroll(context.Background(), owner, championship.ID, uuid.New())
	_, unknownChampionshipErr := env.useCase.Enroll(context.Background(), owner, uuid.New(), team.ID)

	if !errors.Is(unknownTeamErr, entities.ErrNotFound) || !errors.Is(unknownChampionshipErr, entities.ErrNotFound) {
		t.Errorf("both must be not found: %v / %v", unknownTeamErr, unknownChampionshipErr)
	}
}

func TestEnrollRejectsTheSameTeamTwice(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	owner := newTeamCaller()
	team := env.teams.addTeamOwnedBy(owner.UserID)
	if _, err := env.useCase.Enroll(context.Background(), owner, championship.ID, team.ID); err != nil {
		t.Fatal(err)
	}

	updated, err := env.useCase.Enroll(context.Background(), owner, championship.ID, team.ID)

	if !errors.Is(err, entities.ErrTeamAlreadyEnrolled) || updated != nil {
		t.Errorf("want ErrTeamAlreadyEnrolled and nil championship, got %v / %v", err, updated)
	}
}

func TestEnrollStopsWhenAllSlotsAreTaken(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 2)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)
	owner := newTeamCaller()
	extraTeam := env.teams.addTeamOwnedBy(owner.UserID)

	updated, err := env.useCase.Enroll(context.Background(), owner, championship.ID, extraTeam.ID)

	if !errors.Is(err, entities.ErrChampionshipFull) || updated != nil {
		t.Errorf("want ErrChampionshipFull and nil championship, got %v / %v", err, updated)
	}
}

func TestEnrollIsClosedOnceTheChampionshipStarted(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)
	if _, err := env.useCase.Start(context.Background(), championship.ID); err != nil {
		t.Fatal(err)
	}
	owner := newTeamCaller()
	lateTeam := env.teams.addTeamOwnedBy(owner.UserID)

	updated, err := env.useCase.Enroll(context.Background(), owner, championship.ID, lateTeam.ID)

	if !errors.Is(err, entities.ErrChampionshipNotOpen) || updated != nil {
		t.Errorf("want ErrChampionshipNotOpen and nil championship, got %v / %v", err, updated)
	}
}

func TestUnenrollFreesTheSlot(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	owner := newTeamCaller()
	team := env.teams.addTeamOwnedBy(owner.UserID)
	if _, err := env.useCase.Enroll(context.Background(), owner, championship.ID, team.ID); err != nil {
		t.Fatal(err)
	}

	err := env.useCase.Unenroll(context.Background(), owner, championship.ID, team.ID)
	again := env.useCase.Unenroll(context.Background(), owner, championship.ID, team.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, _ := env.useCase.GetByID(context.Background(), championship.ID)
	if stored.SlotsTaken != 0 {
		t.Errorf("the slot must be free, got %d taken", stored.SlotsTaken)
	}
	if !errors.Is(again, entities.ErrTeamNotEnrolled) {
		t.Errorf("a second unenroll must fail: %v", again)
	}
}

func TestUnenrollIsForbiddenForStrangersAndClosedAfterTheStart(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	firstTeam := env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)

	strangerErr := env.useCase.Unenroll(context.Background(), newTeamCaller(), championship.ID, firstTeam.ID)
	env.useCase.Start(context.Background(), championship.ID)
	ownerOfFirst := &entities.Caller{UserID: firstTeam.OwnerID, Role: entities.RoleTeam}
	afterStartErr := env.useCase.Unenroll(context.Background(), ownerOfFirst, championship.ID, firstTeam.ID)

	if !errors.Is(strangerErr, entities.ErrForbidden) {
		t.Errorf("a stranger must be forbidden: %v", strangerErr)
	}
	if !errors.Is(afterStartErr, entities.ErrChampionshipNotOpen) {
		t.Errorf("unenrolling after the start must fail: %v", afterStartErr)
	}
}

func TestStartNeedsAtLeastTwoTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	env.enrollNewTeam(t, championship)

	started, err := env.useCase.Start(context.Background(), championship.ID)

	if !errors.Is(err, entities.ErrInvalidState) || started != nil {
		t.Errorf("want ErrInvalidState and nil championship, got %v / %v", err, started)
	}
}

func TestStartAndFinishFollowTheLifecycle(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	env.enrollNewTeam(t, championship)
	env.enrollNewTeam(t, championship)

	finishTooEarly, finishTooEarlyErr := env.useCase.Finish(context.Background(), championship.ID)
	started, startErr := env.useCase.Start(context.Background(), championship.ID)
	startedAgain, startAgainErr := env.useCase.Start(context.Background(), championship.ID)
	finished, finishErr := env.useCase.Finish(context.Background(), championship.ID)

	if !errors.Is(finishTooEarlyErr, entities.ErrInvalidState) || finishTooEarly != nil {
		t.Errorf("an open championship cannot finish: %v / %v", finishTooEarlyErr, finishTooEarly)
	}
	if startErr != nil || started.Status != entities.ChampionshipInProgress {
		t.Errorf("start failed: %+v / %v", started, startErr)
	}
	if !errors.Is(startAgainErr, entities.ErrInvalidState) || startedAgain != nil {
		t.Errorf("starting twice must fail: %v / %v", startAgainErr, startedAgain)
	}
	if finishErr != nil || finished.Status != entities.ChampionshipFinished {
		t.Errorf("finish failed: %+v / %v", finished, finishErr)
	}
}

func TestListTeamsReturnsTheEnrolledTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	first := env.enrollNewTeam(t, championship)
	second := env.enrollNewTeam(t, championship)

	enrolled, err := env.useCase.ListTeams(context.Background(), championship.ID)

	if err != nil || len(enrolled) != 2 {
		t.Fatalf("want 2 teams, got %d (err=%v)", len(enrolled), err)
	}
	ids := map[uuid.UUID]bool{enrolled[0].ID: true, enrolled[1].ID: true}
	if !ids[first.ID] || !ids[second.ID] {
		t.Errorf("unexpected teams: %+v", enrolled)
	}
}

func TestTeamIsEnrolledAnswersForEnrolledAndOtherTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.createChampionship(t, 10)
	enrolled := env.enrollNewTeam(t, championship)

	yes, yesErr := env.useCase.TeamIsEnrolled(context.Background(), championship.ID, enrolled.ID)
	no, noErr := env.useCase.TeamIsEnrolled(context.Background(), championship.ID, uuid.New())

	if yesErr != nil || noErr != nil || !yes || no {
		t.Errorf("want true/false, got %v/%v (errs %v / %v)", yes, no, yesErr, noErr)
	}
}
