package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"futhub/notification/internal/security"
)

const testSecret = "test-secret-with-at-least-32-characters-long"

type fakePremium struct{ premiumUsers map[string]bool }

func (f fakePremium) IsPremium(_ context.Context, principal security.Principal) (bool, error) {
	return f.premiumUsers[principal.UserID], nil
}

func tokenFor(t *testing.T, userID, role string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID, "role": role, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

type httpFixture struct {
	server    *httptest.Server
	processor *Processor
	store     *RedisStore
	redis     interface{ Close() }
	premiumID string
	freeID    string
}

func newHTTPFixture(t *testing.T) *httpFixture {
	t.Helper()
	store, redisServer := newTestStore(t)
	hub := NewHub()
	premiumID, freeID := uuid.NewString(), uuid.NewString()

	shutdown, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	handler := NewHandler(shutdown, store, hub, fakePremium{premiumUsers: map[string]bool{premiumID: true}},
		security.NewTokenParser(testSecret), quietLogger())
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &httpFixture{
		server: server, processor: NewProcessor(store, hub, quietLogger()), store: store,
		redis: redisServer, premiumID: premiumID, freeID: freeID,
	}
}

func (f *httpFixture) get(t *testing.T, path, token string) *http.Response {
	t.Helper()
	request, _ := http.NewRequest(http.MethodGet, f.server.URL+path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func (f *httpFixture) dial(t *testing.T, matchID, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(f.server.URL, "http") + "/matches/" + matchID + "/stream?token=" + token
	return websocket.DefaultDialer.Dial(url, nil)
}

func readUpdate(t *testing.T, connection *websocket.Conn) Update {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	var update Update
	if err := connection.ReadJSON(&update); err != nil {
		t.Fatalf("could not read update: %v", err)
	}
	return update
}

func TestStreamRejectsMissingTokenAndFreeUsers(t *testing.T) {
	fixture := newHTTPFixture(t)
	matchID := uuid.NewString()
	if err := fixture.processor.Process(context.Background(), startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}

	_, withoutToken, _ := fixture.dial(t, matchID, "")
	_, asFreeUser, _ := fixture.dial(t, matchID, tokenFor(t, fixture.freeID, "USER"))

	if withoutToken.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", withoutToken.StatusCode)
	}
	if asFreeUser.StatusCode != http.StatusForbidden {
		t.Errorf("free user: status = %d, want 403", asFreeUser.StatusCode)
	}
}

func TestStreamOfAMatchThatIsNotLiveIs404(t *testing.T) {
	fixture := newHTTPFixture(t)

	_, response, _ := fixture.dial(t, uuid.NewString(), tokenFor(t, fixture.premiumID, "USER"))

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", response.StatusCode)
	}
}

func TestPremiumUserReceivesStateThenEachUpdateThenTheEnd(t *testing.T) {
	fixture := newHTTPFixture(t)
	ctx := context.Background()
	matchID, goalID := uuid.NewString(), uuid.NewString()
	if err := fixture.processor.Process(ctx, startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}

	connection, _, err := fixture.dial(t, matchID, tokenFor(t, fixture.premiumID, "USER"))
	if err != nil {
		t.Fatalf("premium user could not connect: %v", err)
	}
	defer connection.Close()

	initial := readUpdate(t, connection)
	if initial.Type != UpdateMatchState || initial.Sequence != 1 {
		t.Errorf("first message = %+v, want the current state", initial)
	}

	if err := fixture.processor.Process(ctx, goalMessage(t, matchID, goalID, 2, 1, 0)); err != nil {
		t.Fatal(err)
	}
	goal := readUpdate(t, connection)
	if goal.Type != MessageMatchEventRegistered || goal.Sequence != 2 {
		t.Errorf("second message = %+v, want the goal", goal)
	}
	var payload struct{ Score Score }
	_ = json.Unmarshal(goal.Payload, &payload)
	if payload.Score.Home != 1 {
		t.Errorf("goal payload score = %+v", payload.Score)
	}

	if err := fixture.processor.Process(ctx, finishedMessage(t, matchID, 3, 1, 0)); err != nil {
		t.Fatal(err)
	}
	if final := readUpdate(t, connection); final.Type != MessageMatchFinished {
		t.Errorf("third message = %+v, want MatchFinished", final)
	}
	_ = connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = connection.ReadMessage()
	if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Errorf("after the match ends the socket must close normally, got %v", err)
	}
}

func TestFreeUserCanReadTheSummaryWithTheAuthorizationHeader(t *testing.T) {
	fixture := newHTTPFixture(t)
	matchID := uuid.NewString()
	if err := fixture.processor.Process(context.Background(), startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}

	response := fixture.get(t, "/matches/"+matchID+"/summary", tokenFor(t, fixture.freeID, "USER"))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (free users can read the summary)", response.StatusCode)
	}
	var summary Summary
	_ = json.NewDecoder(response.Body).Decode(&summary)
	if summary.MatchID != matchID || summary.HomeTeam.Name != "Home FC" {
		t.Errorf("unexpected summary: %+v", summary)
	}
}

func TestSummaryOfAFinishedOrUnknownMatchIs404(t *testing.T) {
	fixture := newHTTPFixture(t)
	ctx := context.Background()
	finishedID := uuid.NewString()
	for _, step := range []Envelope{startedMessage(t, finishedID), finishedMessage(t, finishedID, 2, 0, 0)} {
		if err := fixture.processor.Process(ctx, step); err != nil {
			t.Fatal(err)
		}
	}
	token := tokenFor(t, fixture.freeID, "USER")

	for name, matchID := range map[string]string{"finished": finishedID, "unknown": uuid.NewString()} {
		response := fixture.get(t, "/matches/"+matchID+"/summary", token)
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", name, response.StatusCode)
		}
	}
}

func TestSummaryRequiresAValidToken(t *testing.T) {
	fixture := newHTTPFixture(t)

	response := fixture.get(t, "/matches/"+uuid.NewString()+"/summary", "")

	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", response.StatusCode)
	}
}

func TestReadsReturnAClear503WhenRedisIsDown(t *testing.T) {
	fixture := newHTTPFixture(t)
	fixture.redis.Close()

	response := fixture.get(t, "/matches/"+uuid.NewString()+"/summary", tokenFor(t, fixture.freeID, "USER"))

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", response.StatusCode)
	}
	var body struct{ Error struct{ Code string } }
	_ = json.NewDecoder(response.Body).Decode(&body)
	if body.Error.Code != "cache_unavailable" {
		t.Errorf("error code = %q, want cache_unavailable", body.Error.Code)
	}
}

func TestSnapshotJobRefreshesTheSummaryOfLiveMatches(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	processor := NewProcessor(store, NewHub(), quietLogger())
	matchID := uuid.NewString()
	for _, step := range []Envelope{startedMessage(t, matchID), goalMessage(t, matchID, uuid.NewString(), 2, 1, 0)} {
		if err := processor.Process(ctx, step); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := store.LoadSummary(ctx, matchID)

	NewSnapshotJob(store, time.Minute, quietLogger()).RunOnce(ctx)

	after, _ := store.LoadSummary(ctx, matchID)
	if before.Score.Home != 0 || after.Score.Home != 1 || len(after.Highlights) != 1 {
		t.Errorf("summary was not refreshed: before=%+v after=%+v", before, after)
	}
}

func TestHubDropsASlowSubscriberInsteadOfBlocking(t *testing.T) {
	hub := NewHub()
	matchID := uuid.NewString()
	slow := hub.Subscribe(matchID)

	for sequence := 1; sequence <= subscriberBuffer+1; sequence++ {
		hub.Publish(Update{MatchID: matchID, Sequence: sequence})
	}

	if hub.SubscriberCount(matchID) != 0 {
		t.Error("a subscriber that does not read must be dropped")
	}
	slow.Close()
}
