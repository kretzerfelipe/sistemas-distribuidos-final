package matches

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type inMemoryRepository struct {
	matchesByID map[uuid.UUID]entities.Match
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{matchesByID: map[uuid.UUID]entities.Match{}}
}

func (r *inMemoryRepository) Insert(_ context.Context, match *entities.Match) error {
	r.matchesByID[match.ID] = *match
	return nil
}

func (r *inMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Match, error) {
	match, found := r.matchesByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &match, nil
}

func (r *inMemoryRepository) ListByChampionship(_ context.Context, championshipID uuid.UUID) ([]entities.Match, error) {
	found := []entities.Match{}
	for _, match := range r.matchesByID {
		if match.ChampionshipID == championshipID {
			found = append(found, match)
		}
	}
	return found, nil
}

func (r *inMemoryRepository) Reschedule(_ context.Context, id uuid.UUID, scheduledAt, _ time.Time) error {
	match := r.matchesByID[id]
	matchIsNotScheduled := match.Status != entities.MatchScheduled
	if matchIsNotScheduled {
		return entities.ErrInvalidState
	}
	match.ScheduledAt = scheduledAt
	r.matchesByID[id] = match
	return nil
}

func (r *inMemoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	matchIsNotScheduled := r.matchesByID[id].Status != entities.MatchScheduled
	if matchIsNotScheduled {
		return entities.ErrInvalidState
	}
	delete(r.matchesByID, id)
	return nil
}

func (r *inMemoryRepository) Transition(_ context.Context, id uuid.UUID, from, to entities.MatchStatus, homeScore, awayScore int, _ time.Time) error {
	match := r.matchesByID[id]
	matchIsNotInTheExpectedStatus := match.Status != from
	if matchIsNotInTheExpectedStatus {
		return entities.ErrInvalidState
	}
	match.Status = to
	match.HomeScore = homeScore
	match.AwayScore = awayScore
	r.matchesByID[id] = match
	return nil
}

type fakeChampionships struct {
	byID     map[uuid.UUID]*entities.Championship
	enrolled map[uuid.UUID]map[uuid.UUID]bool
}

func (f *fakeChampionships) add(status entities.ChampionshipStatus, teamIDs ...uuid.UUID) *entities.Championship {
	championship := &entities.Championship{ID: uuid.New(), Name: "Copa", Status: status}
	f.byID[championship.ID] = championship
	f.enrolled[championship.ID] = map[uuid.UUID]bool{}
	for _, teamID := range teamIDs {
		f.enrolled[championship.ID][teamID] = true
	}
	return championship
}

func (f *fakeChampionships) GetByID(_ context.Context, id uuid.UUID) (*entities.Championship, error) {
	championship, found := f.byID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	copyOfChampionship := *championship
	return &copyOfChampionship, nil
}

func (f *fakeChampionships) TeamIsEnrolled(_ context.Context, championshipID, teamID uuid.UUID) (bool, error) {
	return f.enrolled[championshipID][teamID], nil
}

type testEnvironment struct {
	useCase       UseCase
	repo          *inMemoryRepository
	championships *fakeChampionships
	homeTeamID    uuid.UUID
	awayTeamID    uuid.UUID
}

func newTestEnvironment() testEnvironment {
	repo := newInMemoryRepository()
	championships := &fakeChampionships{
		byID:     map[uuid.UUID]*entities.Championship{},
		enrolled: map[uuid.UUID]map[uuid.UUID]bool{},
	}
	return testEnvironment{
		useCase:       newUseCase(repo, championships),
		repo:          repo,
		championships: championships,
		homeTeamID:    uuid.New(),
		awayTeamID:    uuid.New(),
	}
}

func (e testEnvironment) newInput(championship *entities.Championship) entities.CreateMatchInput {
	return entities.CreateMatchInput{
		ChampionshipID: championship.ID,
		HomeTeamID:     e.homeTeamID,
		AwayTeamID:     e.awayTeamID,
		ScheduledAt:    time.Date(2027, 3, 1, 15, 0, 0, 0, time.UTC),
	}
}

func (e testEnvironment) createMatchIn(t *testing.T, status entities.ChampionshipStatus) (*entities.Championship, *entities.Match) {
	t.Helper()
	championship := e.championships.add(status, e.homeTeamID, e.awayTeamID)
	match, err := e.useCase.Create(context.Background(), e.newInput(championship))
	if err != nil {
		t.Fatal(err)
	}
	return championship, match
}

func TestCreateSchedulesAMatchBetweenTwoEnrolledTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeamID, env.awayTeamID)

	match, err := env.useCase.Create(context.Background(), env.newInput(championship))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match.Status != entities.MatchScheduled || match.HomeScore != 0 || match.AwayScore != 0 {
		t.Errorf("unexpected match: %+v", match)
	}
}

func TestCreateRejectsATeamPlayingAgainstItself(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeamID)
	input := env.newInput(championship)
	input.AwayTeamID = input.HomeTeamID

	match, err := env.useCase.Create(context.Background(), input)

	if !errors.Is(err, entities.ErrInvalidInput) || match != nil {
		t.Errorf("want ErrInvalidInput and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsAMissingSchedule(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeamID, env.awayTeamID)
	input := env.newInput(championship)
	input.ScheduledAt = time.Time{}

	match, err := env.useCase.Create(context.Background(), input)

	if !errors.Is(err, entities.ErrInvalidInput) || match != nil {
		t.Errorf("want ErrInvalidInput and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsTeamsThatAreNotEnrolledInTheChampionship(t *testing.T) {
	env := newTestEnvironment()
	onlyHomeIsEnrolled := env.championships.add(entities.ChampionshipOpen, env.homeTeamID)
	noneIsEnrolled := env.championships.add(entities.ChampionshipOpen)

	for name, championship := range map[string]*entities.Championship{"only home": onlyHomeIsEnrolled, "none": noneIsEnrolled} {
		t.Run(name, func(t *testing.T) {
			match, err := env.useCase.Create(context.Background(), env.newInput(championship))

			if !errors.Is(err, entities.ErrTeamNotEnrolled) || match != nil {
				t.Errorf("want ErrTeamNotEnrolled and nil match, got %v / %v", err, match)
			}
		})
	}
}

func TestCreateRejectsAFinishedChampionship(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipFinished, env.homeTeamID, env.awayTeamID)

	match, err := env.useCase.Create(context.Background(), env.newInput(championship))

	if !errors.Is(err, entities.ErrInvalidState) || match != nil {
		t.Errorf("want ErrInvalidState and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsAnUnknownChampionship(t *testing.T) {
	env := newTestEnvironment()
	unknown := &entities.Championship{ID: uuid.New()}

	match, err := env.useCase.Create(context.Background(), env.newInput(unknown))

	if !errors.Is(err, entities.ErrNotFound) || match != nil {
		t.Errorf("want ErrNotFound and nil match, got %v / %v", err, match)
	}
}

func TestStartNeedsTheChampionshipInProgress(t *testing.T) {
	env := newTestEnvironment()
	_, matchInOpenChampionship := env.createMatchIn(t, entities.ChampionshipOpen)

	started, err := env.useCase.Start(context.Background(), matchInOpenChampionship.ID)

	if !errors.Is(err, entities.ErrInvalidState) || started != nil {
		t.Errorf("want ErrInvalidState and nil match, got %v / %v", err, started)
	}
}

func TestStartMovesTheMatchToInProgressOnlyOnce(t *testing.T) {
	env := newTestEnvironment()
	_, match := env.createMatchIn(t, entities.ChampionshipInProgress)

	started, err := env.useCase.Start(context.Background(), match.ID)
	again, againErr := env.useCase.Start(context.Background(), match.ID)

	if err != nil || started.Status != entities.MatchInProgress {
		t.Errorf("start failed: %+v / %v", started, err)
	}
	if !errors.Is(againErr, entities.ErrInvalidState) || again != nil {
		t.Errorf("starting twice must fail: %v / %v", againErr, again)
	}
}

func TestFinishRecordsTheFinalScore(t *testing.T) {
	env := newTestEnvironment()
	_, match := env.createMatchIn(t, entities.ChampionshipInProgress)
	if _, err := env.useCase.Start(context.Background(), match.ID); err != nil {
		t.Fatal(err)
	}

	finished, err := env.useCase.Finish(context.Background(), match.ID, entities.FinishMatchInput{HomeScore: 3, AwayScore: 1})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finished.Status != entities.MatchFinished || finished.HomeScore != 3 || finished.AwayScore != 1 {
		t.Errorf("unexpected match: %+v", finished)
	}
}

func TestFinishRejectsNegativeScoresAndMatchesThatDidNotStart(t *testing.T) {
	env := newTestEnvironment()
	_, notStarted := env.createMatchIn(t, entities.ChampionshipInProgress)
	_, started := env.createMatchIn(t, entities.ChampionshipInProgress)
	if _, err := env.useCase.Start(context.Background(), started.ID); err != nil {
		t.Fatal(err)
	}

	beforeStart, beforeStartErr := env.useCase.Finish(context.Background(), notStarted.ID, entities.FinishMatchInput{HomeScore: 1})
	negative, negativeErr := env.useCase.Finish(context.Background(), started.ID, entities.FinishMatchInput{HomeScore: -1})

	if !errors.Is(beforeStartErr, entities.ErrInvalidState) || beforeStart != nil {
		t.Errorf("a match that did not start cannot finish: %v / %v", beforeStartErr, beforeStart)
	}
	if !errors.Is(negativeErr, entities.ErrInvalidInput) || negative != nil {
		t.Errorf("negative scores must be rejected: %v / %v", negativeErr, negative)
	}
}

func TestRescheduleAndDeleteOnlyWorkForScheduledMatches(t *testing.T) {
	env := newTestEnvironment()
	_, scheduled := env.createMatchIn(t, entities.ChampionshipInProgress)
	_, started := env.createMatchIn(t, entities.ChampionshipInProgress)
	if _, err := env.useCase.Start(context.Background(), started.ID); err != nil {
		t.Fatal(err)
	}
	newDate := time.Date(2027, 4, 2, 18, 0, 0, 0, time.UTC)

	moved, moveErr := env.useCase.Reschedule(context.Background(), scheduled.ID, entities.RescheduleMatchInput{ScheduledAt: newDate})
	movedStarted, moveStartedErr := env.useCase.Reschedule(context.Background(), started.ID, entities.RescheduleMatchInput{ScheduledAt: newDate})
	deleteStartedErr := env.useCase.Delete(context.Background(), started.ID)
	deleteScheduledErr := env.useCase.Delete(context.Background(), scheduled.ID)

	if moveErr != nil || !moved.ScheduledAt.Equal(newDate) {
		t.Errorf("a scheduled match can be moved: %+v / %v", moved, moveErr)
	}
	if !errors.Is(moveStartedErr, entities.ErrInvalidState) || movedStarted != nil {
		t.Errorf("a started match cannot be moved: %v / %v", moveStartedErr, movedStarted)
	}
	if !errors.Is(deleteStartedErr, entities.ErrInvalidState) {
		t.Errorf("a started match cannot be deleted: %v", deleteStartedErr)
	}
	if deleteScheduledErr != nil {
		t.Errorf("a scheduled match can be deleted: %v", deleteScheduledErr)
	}
}

func TestRescheduleRejectsAMissingDate(t *testing.T) {
	env := newTestEnvironment()
	_, match := env.createMatchIn(t, entities.ChampionshipOpen)

	moved, err := env.useCase.Reschedule(context.Background(), match.ID, entities.RescheduleMatchInput{})

	if !errors.Is(err, entities.ErrInvalidInput) || moved != nil {
		t.Errorf("want ErrInvalidInput and nil match, got %v / %v", err, moved)
	}
}

func TestGetByIDAndListByChampionship(t *testing.T) {
	env := newTestEnvironment()
	championship, match := env.createMatchIn(t, entities.ChampionshipOpen)

	found, getErr := env.useCase.GetByID(context.Background(), match.ID)
	missing, missingErr := env.useCase.GetByID(context.Background(), uuid.New())
	listed, listErr := env.useCase.ListByChampionship(context.Background(), championship.ID)
	_, unknownChampionshipErr := env.useCase.ListByChampionship(context.Background(), uuid.New())

	if getErr != nil || found.ID != match.ID {
		t.Errorf("get failed: %+v / %v", found, getErr)
	}
	if !errors.Is(missingErr, entities.ErrNotFound) || missing != nil {
		t.Errorf("want ErrNotFound and nil match, got %v / %v", missingErr, missing)
	}
	if listErr != nil || len(listed) != 1 {
		t.Errorf("want one match, got %d (err=%v)", len(listed), listErr)
	}
	if !errors.Is(unknownChampionshipErr, entities.ErrNotFound) {
		t.Errorf("an unknown championship must be not found: %v", unknownChampionshipErr)
	}
}
