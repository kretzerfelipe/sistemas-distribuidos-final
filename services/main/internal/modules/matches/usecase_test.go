package matches

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
)

type inMemoryRepository struct {
	matchesByID   map[uuid.UUID]entities.Match
	eventsByMatch map[uuid.UUID][]entities.MatchEvent
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		matchesByID:   map[uuid.UUID]entities.Match{},
		eventsByMatch: map[uuid.UUID][]entities.MatchEvent{},
	}
}

type repositoryState struct {
	matchesByID   map[uuid.UUID]entities.Match
	eventsByMatch map[uuid.UUID][]entities.MatchEvent
}

func (r *inMemoryRepository) snapshot() repositoryState {
	matchesCopy := map[uuid.UUID]entities.Match{}
	for id, match := range r.matchesByID {
		matchesCopy[id] = match
	}
	eventsCopy := map[uuid.UUID][]entities.MatchEvent{}
	for id, events := range r.eventsByMatch {
		eventsCopy[id] = append([]entities.MatchEvent{}, events...)
	}
	return repositoryState{matchesByID: matchesCopy, eventsByMatch: eventsCopy}
}

func (r *inMemoryRepository) restore(state repositoryState) {
	r.matchesByID = state.matchesByID
	r.eventsByMatch = state.eventsByMatch
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

func (r *inMemoryRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	return r.FindByID(ctx, id)
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

func (r *inMemoryRepository) Transition(_ context.Context, id uuid.UUID, from, to entities.MatchStatus, _ time.Time) error {
	match := r.matchesByID[id]
	matchIsNotInTheExpectedStatus := match.Status != from
	if matchIsNotInTheExpectedStatus {
		return entities.ErrInvalidState
	}
	match.Status = to
	r.matchesByID[id] = match
	return nil
}

func (r *inMemoryRepository) AdvanceSequence(_ context.Context, id uuid.UUID, homeDelta, awayDelta int, _ time.Time) (*entities.MatchProgress, error) {
	match, found := r.matchesByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	match.Sequence++
	match.HomeScore += homeDelta
	match.AwayScore += awayDelta
	r.matchesByID[id] = match
	return &entities.MatchProgress{Sequence: match.Sequence, HomeScore: match.HomeScore, AwayScore: match.AwayScore}, nil
}

func (r *inMemoryRepository) InsertEvent(_ context.Context, event *entities.MatchEvent) error {
	r.eventsByMatch[event.MatchID] = append(r.eventsByMatch[event.MatchID], *event)
	return nil
}

func (r *inMemoryRepository) FindEvent(_ context.Context, matchID, eventID uuid.UUID) (*entities.MatchEvent, error) {
	for _, event := range r.eventsByMatch[matchID] {
		if event.ID == eventID {
			return &event, nil
		}
	}
	return nil, entities.ErrNotFound
}

func (r *inMemoryRepository) ListEvents(_ context.Context, matchID uuid.UUID) ([]entities.MatchEvent, error) {
	events := append([]entities.MatchEvent{}, r.eventsByMatch[matchID]...)
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })
	return events, nil
}

func (r *inMemoryRepository) DeleteEvent(_ context.Context, matchID, eventID uuid.UUID) error {
	remaining := []entities.MatchEvent{}
	wasDeleted := false
	for _, event := range r.eventsByMatch[matchID] {
		if event.ID == eventID {
			wasDeleted = true
			continue
		}
		remaining = append(remaining, event)
	}
	if !wasDeleted {
		return entities.ErrNotFound
	}
	r.eventsByMatch[matchID] = remaining
	return nil
}

func (r *inMemoryRepository) PlayerHasRedCard(_ context.Context, matchID, playerID uuid.UUID) (bool, error) {
	for _, event := range r.eventsByMatch[matchID] {
		playerReceivedARedCard := event.Type == entities.EventRedCard && event.PlayerID != nil && *event.PlayerID == playerID
		if playerReceivedARedCard {
			return true, nil
		}
	}
	return false, nil
}

type rollbackTransactions struct {
	repo *inMemoryRepository
}

func (t *rollbackTransactions) WithTx(ctx context.Context, work func(ctx context.Context) error) error {
	before := t.repo.snapshot()
	err := work(ctx)
	if err != nil {
		t.repo.restore(before)
	}
	return err
}

type recordingOutbox struct {
	added     []outbox.Event
	failsWith error
}

func (o *recordingOutbox) Add(_ context.Context, event outbox.Event) error {
	if o.failsWith != nil {
		return o.failsWith
	}
	o.added = append(o.added, event)
	return nil
}

func (o *recordingOutbox) last(t *testing.T) outbox.Event {
	t.Helper()
	if len(o.added) == 0 {
		t.Fatal("no message was added to the outbox")
	}
	return o.added[len(o.added)-1]
}

type fakeChampionships struct {
	byID     map[uuid.UUID]*entities.Championship
	enrolled map[uuid.UUID]map[uuid.UUID]bool
}

func (f *fakeChampionships) add(status entities.ChampionshipStatus, teamIDs ...uuid.UUID) *entities.Championship {
	championship := &entities.Championship{ID: uuid.New(), Name: "Copa Furb", Status: status}
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

type fakeTeams struct {
	byID    map[uuid.UUID]entities.Team
	rosters map[uuid.UUID]map[uuid.UUID]bool
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

func (f *fakeTeams) PlayerBelongsToTeam(_ context.Context, teamID, playerID uuid.UUID) (bool, error) {
	return f.rosters[teamID][playerID], nil
}

type fakePlayers struct {
	byID map[uuid.UUID]entities.Player
}

func (f *fakePlayers) GetByID(_ context.Context, id uuid.UUID) (*entities.Player, error) {
	player, found := f.byID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &player, nil
}

type testEnvironment struct {
	useCase       UseCase
	repo          *inMemoryRepository
	outbox        *recordingOutbox
	championships *fakeChampionships
	homeTeam      entities.Team
	awayTeam      entities.Team
	homePlayer    entities.Player
	otherHomeMate entities.Player
	awayPlayer    entities.Player
}

func newTestEnvironment() testEnvironment {
	repo := newInMemoryRepository()
	recorder := &recordingOutbox{}
	championships := &fakeChampionships{
		byID:     map[uuid.UUID]*entities.Championship{},
		enrolled: map[uuid.UUID]map[uuid.UUID]bool{},
	}

	homeTeam := entities.Team{ID: uuid.New(), Name: "Home FC"}
	awayTeam := entities.Team{ID: uuid.New(), Name: "Away FC"}
	homePlayer := entities.Player{ID: uuid.New(), Name: "Pele"}
	otherHomeMate := entities.Player{ID: uuid.New(), Name: "Garrincha"}
	awayPlayer := entities.Player{ID: uuid.New(), Name: "Maradona"}

	teams := &fakeTeams{
		byID: map[uuid.UUID]entities.Team{homeTeam.ID: homeTeam, awayTeam.ID: awayTeam},
		rosters: map[uuid.UUID]map[uuid.UUID]bool{
			homeTeam.ID: {homePlayer.ID: true, otherHomeMate.ID: true},
			awayTeam.ID: {awayPlayer.ID: true},
		},
	}
	players := &fakePlayers{byID: map[uuid.UUID]entities.Player{
		homePlayer.ID: homePlayer, otherHomeMate.ID: otherHomeMate, awayPlayer.ID: awayPlayer,
	}}

	return testEnvironment{
		useCase:       newUseCase(repo, championships, teams, players, &rollbackTransactions{repo: repo}, recorder),
		repo:          repo,
		outbox:        recorder,
		championships: championships,
		homeTeam:      homeTeam,
		awayTeam:      awayTeam,
		homePlayer:    homePlayer,
		otherHomeMate: otherHomeMate,
		awayPlayer:    awayPlayer,
	}
}

func (e testEnvironment) newInput(championship *entities.Championship) entities.CreateMatchInput {
	return entities.CreateMatchInput{
		ChampionshipID: championship.ID,
		HomeTeamID:     e.homeTeam.ID,
		AwayTeamID:     e.awayTeam.ID,
		ScheduledAt:    time.Date(2027, 3, 1, 15, 0, 0, 0, time.UTC),
	}
}

func (e testEnvironment) createMatchIn(t *testing.T, status entities.ChampionshipStatus) (*entities.Championship, *entities.Match) {
	t.Helper()
	championship := e.championships.add(status, e.homeTeam.ID, e.awayTeam.ID)
	match, err := e.useCase.Create(context.Background(), e.newInput(championship))
	if err != nil {
		t.Fatal(err)
	}
	return championship, match
}

func (e testEnvironment) createLiveMatch(t *testing.T) *entities.Match {
	t.Helper()
	_, match := e.createMatchIn(t, entities.ChampionshipInProgress)
	if _, err := e.useCase.Start(context.Background(), match.ID); err != nil {
		t.Fatal(err)
	}
	return match
}

func (e testEnvironment) register(t *testing.T, matchID uuid.UUID, in entities.RegisterMatchEventInput) *entities.MatchEvent {
	t.Helper()
	event, err := e.useCase.RegisterEvent(context.Background(), matchID, in)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func playerID(player entities.Player) *uuid.UUID { return &player.ID }

func (e testEnvironment) homeGoalBy(player entities.Player, minute int) entities.RegisterMatchEventInput {
	return entities.RegisterMatchEventInput{Type: entities.EventGoal, Minute: minute, TeamID: e.homeTeam.ID, PlayerID: playerID(player)}
}

func TestCreateSchedulesAMatchBetweenTwoEnrolledTeams(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeam.ID, env.awayTeam.ID)

	match, err := env.useCase.Create(context.Background(), env.newInput(championship))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match.Status != entities.MatchScheduled || match.HomeScore != 0 || match.AwayScore != 0 || match.Sequence != 0 {
		t.Errorf("unexpected match: %+v", match)
	}
	if len(env.outbox.added) != 0 {
		t.Error("scheduling a match must not publish anything")
	}
}

func TestCreateRejectsATeamPlayingAgainstItself(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeam.ID)
	input := env.newInput(championship)
	input.AwayTeamID = input.HomeTeamID

	match, err := env.useCase.Create(context.Background(), input)

	if !errors.Is(err, entities.ErrInvalidInput) || match != nil {
		t.Errorf("want ErrInvalidInput and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsAMissingSchedule(t *testing.T) {
	env := newTestEnvironment()
	championship := env.championships.add(entities.ChampionshipOpen, env.homeTeam.ID, env.awayTeam.ID)
	input := env.newInput(championship)
	input.ScheduledAt = time.Time{}

	match, err := env.useCase.Create(context.Background(), input)

	if !errors.Is(err, entities.ErrInvalidInput) || match != nil {
		t.Errorf("want ErrInvalidInput and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsTeamsThatAreNotEnrolledInTheChampionship(t *testing.T) {
	env := newTestEnvironment()
	onlyHomeIsEnrolled := env.championships.add(entities.ChampionshipOpen, env.homeTeam.ID)
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
	championship := env.championships.add(entities.ChampionshipFinished, env.homeTeam.ID, env.awayTeam.ID)

	match, err := env.useCase.Create(context.Background(), env.newInput(championship))

	if !errors.Is(err, entities.ErrInvalidState) || match != nil {
		t.Errorf("want ErrInvalidState and nil match, got %v / %v", err, match)
	}
}

func TestCreateRejectsAnUnknownChampionship(t *testing.T) {
	env := newTestEnvironment()

	match, err := env.useCase.Create(context.Background(), env.newInput(&entities.Championship{ID: uuid.New()}))

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
	if len(env.outbox.added) != 0 {
		t.Error("nothing may be published when the match does not start")
	}
}

func TestStartMovesTheMatchToInProgressAndPublishesMatchStarted(t *testing.T) {
	env := newTestEnvironment()
	championship, match := env.createMatchIn(t, entities.ChampionshipInProgress)

	started, err := env.useCase.Start(context.Background(), match.ID)

	if err != nil || started.Status != entities.MatchInProgress {
		t.Fatalf("start failed: %+v / %v", started, err)
	}
	message := env.outbox.last(t)
	payload, isMatchStarted := message.Payload.(entities.MatchStartedPayload)
	if !isMatchStarted || message.Type != "MatchStarted" || message.RoutingKey != "match.started" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if payload.MatchID != match.ID || payload.Sequence != 1 || payload.Score != (entities.MatchScore{}) {
		t.Errorf("unexpected payload: %+v", payload)
	}
	if payload.HomeTeam.Name != "Home FC" || payload.AwayTeam.Name != "Away FC" || payload.Championship.ID != championship.ID || payload.Championship.Name != "Copa Furb" {
		t.Errorf("the message must carry the names the live view needs: %+v", payload)
	}
}

func TestStartTwiceFailsAndPublishesOnlyOnce(t *testing.T) {
	env := newTestEnvironment()
	_, match := env.createMatchIn(t, entities.ChampionshipInProgress)
	if _, err := env.useCase.Start(context.Background(), match.ID); err != nil {
		t.Fatal(err)
	}

	again, err := env.useCase.Start(context.Background(), match.ID)

	if !errors.Is(err, entities.ErrInvalidState) || again != nil {
		t.Errorf("starting twice must fail: %v / %v", err, again)
	}
	if len(env.outbox.added) != 1 {
		t.Errorf("only one MatchStarted may be published, got %d messages", len(env.outbox.added))
	}
}

func TestStartRollsBackWhenTheMessageCannotBeStored(t *testing.T) {
	env := newTestEnvironment()
	_, match := env.createMatchIn(t, entities.ChampionshipInProgress)
	env.outbox.failsWith = errors.New("disk full")

	started, err := env.useCase.Start(context.Background(), match.ID)

	if err == nil || started != nil {
		t.Fatalf("start must fail: %v / %v", err, started)
	}
	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	if stored.Status != entities.MatchScheduled || stored.Sequence != 0 {
		t.Errorf("the match must stay untouched, got %+v", stored)
	}
}

func TestRegisterEventCountsAGoalAndPublishesTheScore(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	event := env.register(t, match.ID, env.homeGoalBy(env.homePlayer, 23))

	if event.Sequence != 2 || event.PlayerName != "Pele" || event.Type != entities.EventGoal {
		t.Errorf("unexpected event: %+v", event)
	}
	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	if stored.HomeScore != 1 || stored.AwayScore != 0 {
		t.Errorf("want 1x0, got %dx%d", stored.HomeScore, stored.AwayScore)
	}
	payload, isRegistered := env.outbox.last(t).Payload.(entities.MatchEventRegisteredPayload)
	if !isRegistered || env.outbox.last(t).RoutingKey != "match.event-registered" {
		t.Fatalf("unexpected message: %+v", env.outbox.last(t))
	}
	if payload.Sequence != 2 || payload.Score != (entities.MatchScore{Home: 1}) || payload.Event.ID != event.ID {
		t.Errorf("unexpected payload: %+v", payload)
	}
	if payload.Event.Player == nil || payload.Event.Player.Name != "Pele" || payload.Event.SecondaryPlayer != nil {
		t.Errorf("the message must carry the player name: %+v", payload.Event)
	}
}

func TestRegisterEventGivesGoalsToTheRightTeam(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	env.register(t, match.ID, entities.RegisterMatchEventInput{Type: entities.EventGoal, Minute: 10, TeamID: env.awayTeam.ID})
	env.register(t, match.ID, entities.RegisterMatchEventInput{Type: entities.EventOwnGoal, Minute: 20, TeamID: env.awayTeam.ID})
	env.register(t, match.ID, entities.RegisterMatchEventInput{Type: entities.EventOwnGoal, Minute: 30, TeamID: env.homeTeam.ID})

	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	if stored.HomeScore != 1 || stored.AwayScore != 2 {
		t.Errorf("an own goal counts for the opponent: want 1x2, got %dx%d", stored.HomeScore, stored.AwayScore)
	}
}

func TestRegisterEventKeepsTheScoreForPlaysThatAreNotGoals(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	for _, eventType := range []entities.MatchEventType{
		entities.EventShot, entities.EventSave, entities.EventFoul, entities.EventYellowCard, entities.EventCorner, entities.EventOffside,
	} {
		env.register(t, match.ID, entities.RegisterMatchEventInput{Type: eventType, Minute: 5, TeamID: env.homeTeam.ID})
	}

	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	if stored.HomeScore != 0 || stored.AwayScore != 0 || stored.Sequence != 7 {
		t.Errorf("want 0x0 and sequence 7, got %dx%d and %d", stored.HomeScore, stored.AwayScore, stored.Sequence)
	}
}

func TestRegisterEventNumbersEveryChangeInOrder(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	first := env.register(t, match.ID, env.homeGoalBy(env.homePlayer, 10))
	second := env.register(t, match.ID, env.homeGoalBy(env.otherHomeMate, 50))

	if first.Sequence != 2 || second.Sequence != 3 {
		t.Errorf("want sequences 2 and 3, got %d and %d", first.Sequence, second.Sequence)
	}
}

func TestRegisterEventOnlyWorksWhileTheMatchIsInProgress(t *testing.T) {
	env := newTestEnvironment()
	_, scheduled := env.createMatchIn(t, entities.ChampionshipInProgress)
	finished := env.createLiveMatch(t)
	if _, err := env.useCase.Finish(context.Background(), finished.ID); err != nil {
		t.Fatal(err)
	}

	for name, matchID := range map[string]uuid.UUID{"scheduled": scheduled.ID, "finished": finished.ID} {
		t.Run(name, func(t *testing.T) {
			event, err := env.useCase.RegisterEvent(context.Background(), matchID, env.homeGoalBy(env.homePlayer, 1))

			if !errors.Is(err, entities.ErrInvalidState) || event != nil {
				t.Errorf("want ErrInvalidState and nil event, got %v / %v", err, event)
			}
		})
	}
}

func TestRegisterEventRejectsInvalidInput(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	stranger := uuid.New()

	scenarios := map[string]entities.RegisterMatchEventInput{
		"unknown type":                 {Type: "DANCE", Minute: 10, TeamID: env.homeTeam.ID},
		"negative minute":              {Type: entities.EventGoal, Minute: -1, TeamID: env.homeTeam.ID},
		"minute above maximum":         {Type: entities.EventGoal, Minute: maxEventMinute + 1, TeamID: env.homeTeam.ID},
		"team not in match":            {Type: entities.EventGoal, Minute: 10, TeamID: stranger},
		"substitution without players": {Type: entities.EventSubstitution, Minute: 60, TeamID: env.homeTeam.ID},
		"substitution with one player": {Type: entities.EventSubstitution, Minute: 60, TeamID: env.homeTeam.ID, PlayerID: playerID(env.homePlayer)},
		"substitution with the same player": {Type: entities.EventSubstitution, Minute: 60, TeamID: env.homeTeam.ID,
			PlayerID: playerID(env.homePlayer), SecondaryPlayerID: playerID(env.homePlayer)},
		"secondary player outside a substitution": {Type: entities.EventFoul, Minute: 60, TeamID: env.homeTeam.ID,
			PlayerID: playerID(env.homePlayer), SecondaryPlayerID: playerID(env.otherHomeMate)},
	}

	for name, input := range scenarios {
		t.Run(name, func(t *testing.T) {
			event, err := env.useCase.RegisterEvent(context.Background(), match.ID, input)

			if !errors.Is(err, entities.ErrInvalidInput) || event != nil {
				t.Errorf("want ErrInvalidInput and nil event, got %v / %v", err, event)
			}
		})
	}
	if len(env.outbox.added) != 1 {
		t.Errorf("rejected events must not publish anything, got %d messages", len(env.outbox.added))
	}
}

func TestRegisterEventAcceptsASubstitutionWithTwoPlayers(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	event := env.register(t, match.ID, entities.RegisterMatchEventInput{
		Type: entities.EventSubstitution, Minute: 70, TeamID: env.homeTeam.ID,
		PlayerID: playerID(env.otherHomeMate), SecondaryPlayerID: playerID(env.homePlayer),
	})

	if event.PlayerName != "Garrincha" || event.SecondaryPlayerName != "Pele" {
		t.Errorf("want Garrincha in and Pele out, got %q and %q", event.PlayerName, event.SecondaryPlayerName)
	}
}

func TestRegisterEventRejectsAPlayerFromAnotherTeamOrAnUnknownPlayer(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)

	fromOtherTeam, otherTeamErr := env.useCase.RegisterEvent(context.Background(), match.ID, env.homeGoalBy(env.awayPlayer, 10))
	unknown, unknownErr := env.useCase.RegisterEvent(context.Background(), match.ID,
		entities.RegisterMatchEventInput{Type: entities.EventGoal, Minute: 10, TeamID: env.homeTeam.ID, PlayerID: playerID(entities.Player{ID: uuid.New()})})

	if !errors.Is(otherTeamErr, entities.ErrInvalidInput) || fromOtherTeam != nil {
		t.Errorf("a player of the other team must be rejected: %v / %v", otherTeamErr, fromOtherTeam)
	}
	if !errors.Is(unknownErr, entities.ErrNotFound) || unknown != nil {
		t.Errorf("an unknown player must be not found: %v / %v", unknownErr, unknown)
	}
}

func TestAPlayerWhoWasSentOffHasNoMorePlays(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	env.register(t, match.ID, entities.RegisterMatchEventInput{
		Type: entities.EventRedCard, Minute: 30, TeamID: env.homeTeam.ID, PlayerID: playerID(env.homePlayer)})

	asPlayer, asPlayerErr := env.useCase.RegisterEvent(context.Background(), match.ID, env.homeGoalBy(env.homePlayer, 40))
	asSecondary, asSecondaryErr := env.useCase.RegisterEvent(context.Background(), match.ID, entities.RegisterMatchEventInput{
		Type: entities.EventSubstitution, Minute: 41, TeamID: env.homeTeam.ID,
		PlayerID: playerID(env.otherHomeMate), SecondaryPlayerID: playerID(env.homePlayer)})
	teammate := env.register(t, match.ID, env.homeGoalBy(env.otherHomeMate, 45))

	if !errors.Is(asPlayerErr, entities.ErrInvalidState) || asPlayer != nil {
		t.Errorf("a sent off player cannot score: %v / %v", asPlayerErr, asPlayer)
	}
	if !errors.Is(asSecondaryErr, entities.ErrInvalidState) || asSecondary != nil {
		t.Errorf("a sent off player cannot be substituted: %v / %v", asSecondaryErr, asSecondary)
	}
	if teammate == nil {
		t.Error("a teammate must still be able to play")
	}
}

func TestRegisterEventRollsBackWhenTheMessageCannotBeStored(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	env.outbox.failsWith = errors.New("disk full")

	event, err := env.useCase.RegisterEvent(context.Background(), match.ID, env.homeGoalBy(env.homePlayer, 10))

	if err == nil || event != nil {
		t.Fatalf("registering must fail: %v / %v", err, event)
	}
	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	storedEvents, _ := env.useCase.ListEvents(context.Background(), match.ID)
	if stored.HomeScore != 0 || stored.Sequence != 1 || len(storedEvents) != 0 {
		t.Errorf("event, score and sequence must roll back together: %+v / %d events", stored, len(storedEvents))
	}
}

func TestRemoveEventRevertsTheScoreAndPublishesTheCorrection(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	goal := env.register(t, match.ID, env.homeGoalBy(env.homePlayer, 10))
	env.register(t, match.ID, env.homeGoalBy(env.otherHomeMate, 20))

	err := env.useCase.RemoveEvent(context.Background(), match.ID, goal.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, _ := env.useCase.GetByID(context.Background(), match.ID)
	if stored.HomeScore != 1 {
		t.Errorf("want 1x0 after the removal, got %dx%d", stored.HomeScore, stored.AwayScore)
	}
	message := env.outbox.last(t)
	payload, isRemoved := message.Payload.(entities.MatchEventRemovedPayload)
	if !isRemoved || message.RoutingKey != "match.event-removed" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if payload.EventID != goal.ID || payload.Sequence != 4 || payload.Score != (entities.MatchScore{Home: 1}) {
		t.Errorf("unexpected payload: %+v", payload)
	}
	remaining, _ := env.useCase.ListEvents(context.Background(), match.ID)
	if len(remaining) != 1 {
		t.Errorf("one event must remain, got %d", len(remaining))
	}
}

func TestRemoveEventFailsForUnknownEventsAndMatchesNotInProgress(t *testing.T) {
	env := newTestEnvironment()
	live := env.createLiveMatch(t)
	goal := env.register(t, live.ID, env.homeGoalBy(env.homePlayer, 10))
	_, scheduled := env.createMatchIn(t, entities.ChampionshipInProgress)

	unknownEventErr := env.useCase.RemoveEvent(context.Background(), live.ID, uuid.New())
	scheduledErr := env.useCase.RemoveEvent(context.Background(), scheduled.ID, goal.ID)
	if _, err := env.useCase.Finish(context.Background(), live.ID); err != nil {
		t.Fatal(err)
	}
	finishedErr := env.useCase.RemoveEvent(context.Background(), live.ID, goal.ID)

	if !errors.Is(unknownEventErr, entities.ErrNotFound) {
		t.Errorf("an unknown event must be not found: %v", unknownEventErr)
	}
	if !errors.Is(scheduledErr, entities.ErrInvalidState) || !errors.Is(finishedErr, entities.ErrInvalidState) {
		t.Errorf("only matches in progress allow removals: %v / %v", scheduledErr, finishedErr)
	}
}

func TestFinishPublishesTheComputedFinalScore(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	env.register(t, match.ID, env.homeGoalBy(env.homePlayer, 10))
	env.register(t, match.ID, entities.RegisterMatchEventInput{Type: entities.EventGoal, Minute: 20, TeamID: env.awayTeam.ID})
	env.register(t, match.ID, env.homeGoalBy(env.otherHomeMate, 30))

	finished, err := env.useCase.Finish(context.Background(), match.ID)

	if err != nil || finished.Status != entities.MatchFinished || finished.HomeScore != 2 || finished.AwayScore != 1 {
		t.Fatalf("finish failed: %+v / %v", finished, err)
	}
	message := env.outbox.last(t)
	payload, isFinished := message.Payload.(entities.MatchFinishedPayload)
	if !isFinished || message.RoutingKey != "match.finished" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if payload.Sequence != 5 || payload.Score != (entities.MatchScore{Home: 2, Away: 1}) {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestFinishOnlyWorksForAMatchInProgress(t *testing.T) {
	env := newTestEnvironment()
	_, scheduled := env.createMatchIn(t, entities.ChampionshipInProgress)
	live := env.createLiveMatch(t)
	if _, err := env.useCase.Finish(context.Background(), live.ID); err != nil {
		t.Fatal(err)
	}

	beforeStart, beforeStartErr := env.useCase.Finish(context.Background(), scheduled.ID)
	again, againErr := env.useCase.Finish(context.Background(), live.ID)

	if !errors.Is(beforeStartErr, entities.ErrInvalidState) || beforeStart != nil {
		t.Errorf("a match that did not start cannot finish: %v / %v", beforeStartErr, beforeStart)
	}
	if !errors.Is(againErr, entities.ErrInvalidState) || again != nil {
		t.Errorf("finishing twice must fail: %v / %v", againErr, again)
	}
}

func TestRescheduleAndDeleteOnlyWorkForScheduledMatches(t *testing.T) {
	env := newTestEnvironment()
	_, scheduled := env.createMatchIn(t, entities.ChampionshipInProgress)
	started := env.createLiveMatch(t)
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

func TestListEventsReturnsThePlaysInOrderAndFailsForUnknownMatches(t *testing.T) {
	env := newTestEnvironment()
	match := env.createLiveMatch(t)
	env.register(t, match.ID, env.homeGoalBy(env.homePlayer, 10))
	env.register(t, match.ID, entities.RegisterMatchEventInput{Type: entities.EventYellowCard, Minute: 15, TeamID: env.awayTeam.ID})

	listed, err := env.useCase.ListEvents(context.Background(), match.ID)
	_, unknownErr := env.useCase.ListEvents(context.Background(), uuid.New())

	if err != nil || len(listed) != 2 || listed[0].Sequence >= listed[1].Sequence {
		t.Errorf("want 2 events in order, got %+v (err=%v)", listed, err)
	}
	if !errors.Is(unknownErr, entities.ErrNotFound) {
		t.Errorf("an unknown match must be not found: %v", unknownErr)
	}
}
