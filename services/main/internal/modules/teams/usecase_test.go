package teams

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type inMemoryRepository struct {
	teamsByID       map[uuid.UUID]entities.Team
	teamOfPlayer    map[uuid.UUID]uuid.UUID
	invitationsByID map[uuid.UUID]entities.Invitation
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		teamsByID:       map[uuid.UUID]entities.Team{},
		teamOfPlayer:    map[uuid.UUID]uuid.UUID{},
		invitationsByID: map[uuid.UUID]entities.Invitation{},
	}
}

func (r *inMemoryRepository) InsertTeam(_ context.Context, team *entities.Team) error {
	for _, existing := range r.teamsByID {
		nameIsTaken := strings.EqualFold(existing.Name, team.Name)
		if nameIsTaken {
			return entities.ErrTeamNameTaken
		}
	}
	r.teamsByID[team.ID] = *team
	return nil
}

func (r *inMemoryRepository) FindTeamByID(_ context.Context, id uuid.UUID) (*entities.Team, error) {
	team, found := r.teamsByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &team, nil
}

func (r *inMemoryRepository) FindTeamsByIDs(_ context.Context, ids []uuid.UUID) ([]entities.Team, error) {
	found := []entities.Team{}
	for _, id := range ids {
		if team, exists := r.teamsByID[id]; exists {
			found = append(found, team)
		}
	}
	return found, nil
}

func (r *inMemoryRepository) ListTeams(context.Context) ([]entities.Team, error) {
	all := []entities.Team{}
	for _, team := range r.teamsByID {
		all = append(all, team)
	}
	return all, nil
}

func (r *inMemoryRepository) ListTeamsByOwner(_ context.Context, ownerID uuid.UUID) ([]entities.Team, error) {
	owned := []entities.Team{}
	for _, team := range r.teamsByID {
		if team.OwnerID == ownerID {
			owned = append(owned, team)
		}
	}
	return owned, nil
}

func (r *inMemoryRepository) UpdateTeam(_ context.Context, team *entities.Team) error {
	for id, existing := range r.teamsByID {
		nameIsTakenByAnotherTeam := id != team.ID && strings.EqualFold(existing.Name, team.Name)
		if nameIsTakenByAnotherTeam {
			return entities.ErrTeamNameTaken
		}
	}
	r.teamsByID[team.ID] = *team
	return nil
}

func (r *inMemoryRepository) DeleteTeam(_ context.Context, id uuid.UUID) error {
	_, exists := r.teamsByID[id]
	if !exists {
		return entities.ErrNotFound
	}
	delete(r.teamsByID, id)
	return nil
}

func (r *inMemoryRepository) ListMemberPlayerIDs(_ context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	memberIDs := []uuid.UUID{}
	for playerID, memberTeamID := range r.teamOfPlayer {
		if memberTeamID == teamID {
			memberIDs = append(memberIDs, playerID)
		}
	}
	sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i].String() < memberIDs[j].String() })
	return memberIDs, nil
}

func (r *inMemoryRepository) PlayerIsInTeam(_ context.Context, playerID uuid.UUID) (bool, error) {
	_, isInTeam := r.teamOfPlayer[playerID]
	return isInTeam, nil
}

func (r *inMemoryRepository) PlayerIsMemberOfTeam(_ context.Context, teamID, playerID uuid.UUID) (bool, error) {
	memberTeamID, isInTeam := r.teamOfPlayer[playerID]
	return isInTeam && memberTeamID == teamID, nil
}

func (r *inMemoryRepository) DeleteMembership(_ context.Context, teamID, playerID uuid.UUID) error {
	memberTeamID, isInTeam := r.teamOfPlayer[playerID]
	playerIsNotInThisTeam := !isInTeam || memberTeamID != teamID
	if playerIsNotInThisTeam {
		return entities.ErrPlayerNotInTeam
	}
	delete(r.teamOfPlayer, playerID)
	return nil
}

func (r *inMemoryRepository) DeleteMembershipByPlayer(_ context.Context, playerID uuid.UUID) error {
	_, isInTeam := r.teamOfPlayer[playerID]
	if !isInTeam {
		return entities.ErrPlayerNotInTeam
	}
	delete(r.teamOfPlayer, playerID)
	return nil
}

func (r *inMemoryRepository) FindTeamByPlayer(_ context.Context, playerID uuid.UUID) (*entities.Team, error) {
	teamID, isInTeam := r.teamOfPlayer[playerID]
	if !isInTeam {
		return nil, entities.ErrNotFound
	}
	team := r.teamsByID[teamID]
	return &team, nil
}

func (r *inMemoryRepository) InsertInvitation(_ context.Context, invitation *entities.Invitation) error {
	for _, existing := range r.invitationsByID {
		samePair := existing.TeamID == invitation.TeamID && existing.PlayerID == invitation.PlayerID
		if samePair && existing.Status == entities.InvitationPending {
			return entities.ErrInvitationAlreadyPending
		}
	}
	r.invitationsByID[invitation.ID] = *invitation
	return nil
}

func (r *inMemoryRepository) FindInvitationByID(_ context.Context, id uuid.UUID) (*entities.Invitation, error) {
	invitation, found := r.invitationsByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	team := r.teamsByID[invitation.TeamID]
	invitation.TeamName = team.Name
	return &invitation, nil
}

func (r *inMemoryRepository) ListInvitationsByTeam(_ context.Context, teamID uuid.UUID) ([]entities.Invitation, error) {
	found := []entities.Invitation{}
	for _, invitation := range r.invitationsByID {
		if invitation.TeamID == teamID {
			found = append(found, invitation)
		}
	}
	return found, nil
}

func (r *inMemoryRepository) ListInvitationsByPlayer(_ context.Context, playerID uuid.UUID) ([]entities.Invitation, error) {
	found := []entities.Invitation{}
	for _, invitation := range r.invitationsByID {
		if invitation.PlayerID == playerID {
			found = append(found, invitation)
		}
	}
	return found, nil
}

func (r *inMemoryRepository) UpdateInvitationStatus(_ context.Context, id uuid.UUID, from, to entities.InvitationStatus, _ time.Time) error {
	invitation, found := r.invitationsByID[id]
	invitationIsNotInTheExpectedStatus := !found || invitation.Status != from
	if invitationIsNotInTheExpectedStatus {
		return entities.ErrInvitationNotPending
	}
	invitation.Status = to
	r.invitationsByID[id] = invitation
	return nil
}

func (r *inMemoryRepository) AcceptInvitation(_ context.Context, invitation *entities.Invitation, _ time.Time) error {
	stored, found := r.invitationsByID[invitation.ID]
	invitationIsNotPending := !found || stored.Status != entities.InvitationPending
	if invitationIsNotPending {
		return entities.ErrInvitationNotPending
	}
	_, playerAlreadyHasATeam := r.teamOfPlayer[invitation.PlayerID]
	if playerAlreadyHasATeam {
		return entities.ErrPlayerAlreadyInTeam
	}

	stored.Status = entities.InvitationAccepted
	r.invitationsByID[stored.ID] = stored
	r.teamOfPlayer[invitation.PlayerID] = invitation.TeamID
	for id, other := range r.invitationsByID {
		isAnotherPendingInvitationOfThePlayer := other.PlayerID == invitation.PlayerID && other.Status == entities.InvitationPending
		if isAnotherPendingInvitationOfThePlayer {
			other.Status = entities.InvitationCanceled
			r.invitationsByID[id] = other
		}
	}
	return nil
}

type fakePlayers struct {
	byID     map[uuid.UUID]entities.Player
	byUserID map[uuid.UUID]entities.Player
}

func newFakePlayers() *fakePlayers {
	return &fakePlayers{byID: map[uuid.UUID]entities.Player{}, byUserID: map[uuid.UUID]entities.Player{}}
}

func (f *fakePlayers) addPlayerFor(userID uuid.UUID) entities.Player {
	player := entities.Player{ID: uuid.New(), UserID: userID, Name: "Player " + userID.String()[:4]}
	f.byID[player.ID] = player
	f.byUserID[userID] = player
	return player
}

func (f *fakePlayers) GetByID(_ context.Context, id uuid.UUID) (*entities.Player, error) {
	player, found := f.byID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &player, nil
}

func (f *fakePlayers) GetByUserID(_ context.Context, userID uuid.UUID) (*entities.Player, error) {
	player, found := f.byUserID[userID]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &player, nil
}

func (f *fakePlayers) GetByIDs(_ context.Context, ids []uuid.UUID) ([]entities.Player, error) {
	found := []entities.Player{}
	for _, id := range ids {
		if player, exists := f.byID[id]; exists {
			found = append(found, player)
		}
	}
	return found, nil
}

type testEnvironment struct {
	useCase UseCase
	repo    *inMemoryRepository
	players *fakePlayers
	storage *fakeStorage
}

func newTestEnvironment() testEnvironment {
	repo := newInMemoryRepository()
	players := newFakePlayers()
	photos := newFakeStorage()
	return testEnvironment{useCase: newUseCase(repo, players, photos), repo: repo, players: players, storage: photos}
}

func newTeamCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleTeam}
}

func newUserCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleUser}
}

func newAdminCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}
}

func (e testEnvironment) createTeam(t *testing.T, owner *entities.Caller, name string) *entities.Team {
	t.Helper()
	team, err := e.useCase.Create(context.Background(), owner, entities.SaveTeamInput{Name: name, City: "Blumenau"})
	if err != nil {
		t.Fatal(err)
	}
	return team
}

func (e testEnvironment) newPlayerUser() (*entities.Caller, entities.Player) {
	caller := newUserCaller()
	return caller, e.players.addPlayerFor(caller.UserID)
}

func (e testEnvironment) invite(t *testing.T, owner *entities.Caller, team *entities.Team, player entities.Player) *entities.Invitation {
	t.Helper()
	invitation, err := e.useCase.Invite(context.Background(), owner, team.ID, player.ID)
	if err != nil {
		t.Fatal(err)
	}
	return invitation
}

func TestCreateMakesTheCallerTheOwner(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()

	team := env.createTeam(t, owner, "Furb FC")

	if team.OwnerID != owner.UserID || team.Name != "Furb FC" {
		t.Errorf("unexpected team: %+v", team)
	}
}

func TestCreateRejectsDuplicateNamesIgnoringCase(t *testing.T) {
	env := newTestEnvironment()
	env.createTeam(t, newTeamCaller(), "Furb FC")

	team, err := env.useCase.Create(context.Background(), newTeamCaller(), entities.SaveTeamInput{Name: "FURB fc"})

	if !errors.Is(err, entities.ErrTeamNameTaken) || team != nil {
		t.Errorf("want ErrTeamNameTaken and nil team, got %v / %v", err, team)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	scenarios := map[string]entities.SaveTeamInput{
		"blank name":    {Name: "   "},
		"name too long": {Name: strings.Repeat("a", maxTeamNameLength+1)},
		"city too long": {Name: "Furb FC", City: strings.Repeat("a", maxCityLength+1)},
	}

	for name, input := range scenarios {
		t.Run(name, func(t *testing.T) {
			team, err := newTestEnvironment().useCase.Create(context.Background(), newTeamCaller(), input)

			if !errors.Is(err, entities.ErrInvalidInput) || team != nil {
				t.Errorf("want ErrInvalidInput and nil team, got %v / %v", err, team)
			}
		})
	}
}

func TestListMineReturnsOnlyTheTeamsOfTheCaller(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	env.createTeam(t, owner, "Mine A")
	env.createTeam(t, owner, "Mine B")
	env.createTeam(t, newTeamCaller(), "Someone Else")

	mine, err := env.useCase.ListMine(context.Background(), owner)

	if err != nil || len(mine) != 2 {
		t.Errorf("want 2 teams, got %d (err=%v)", len(mine), err)
	}
}

func TestUpdateIsAllowedForTheOwnerAndTheAdminButNotForOthers(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	input := entities.SaveTeamInput{Name: "Furb United", City: "Itajai"}

	byStranger, strangerErr := env.useCase.Update(context.Background(), newTeamCaller(), team.ID, input)
	byOwner, ownerErr := env.useCase.Update(context.Background(), owner, team.ID, input)
	byAdmin, adminErr := env.useCase.Update(context.Background(), newAdminCaller(), team.ID, entities.SaveTeamInput{Name: "Admin Edit"})

	if !errors.Is(strangerErr, entities.ErrForbidden) || byStranger != nil {
		t.Errorf("a stranger must be forbidden: %v / %v", strangerErr, byStranger)
	}
	if ownerErr != nil || byOwner.Name != "Furb United" || byOwner.City != "Itajai" {
		t.Errorf("the owner must be able to update: %+v / %v", byOwner, ownerErr)
	}
	if adminErr != nil || byAdmin.Name != "Admin Edit" {
		t.Errorf("the admin must be able to update: %+v / %v", byAdmin, adminErr)
	}
}

func TestUpdateRejectsANameUsedByAnotherTeam(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	env.createTeam(t, owner, "Taken")
	team := env.createTeam(t, owner, "Mine")

	updated, err := env.useCase.Update(context.Background(), owner, team.ID, entities.SaveTeamInput{Name: "taken"})

	if !errors.Is(err, entities.ErrTeamNameTaken) || updated != nil {
		t.Errorf("want ErrTeamNameTaken and nil team, got %v / %v", err, updated)
	}
}

func TestDeleteIsAllowedOnlyForTheOwnerOrTheAdmin(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")

	strangerErr := env.useCase.Delete(context.Background(), newTeamCaller(), team.ID)
	ownerErr := env.useCase.Delete(context.Background(), owner, team.ID)
	missingErr := env.useCase.Delete(context.Background(), owner, team.ID)

	if !errors.Is(strangerErr, entities.ErrForbidden) {
		t.Errorf("a stranger must be forbidden, got %v", strangerErr)
	}
	if ownerErr != nil {
		t.Errorf("the owner must be able to delete, got %v", ownerErr)
	}
	if !errors.Is(missingErr, entities.ErrNotFound) {
		t.Errorf("a deleted team must be gone, got %v", missingErr)
	}
}

func TestInviteCreatesAPendingInvitation(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()

	invitation := env.invite(t, owner, team, player)

	if invitation.Status != entities.InvitationPending || invitation.TeamName != "Furb FC" || invitation.PlayerID != player.ID {
		t.Errorf("unexpected invitation: %+v", invitation)
	}
}

func TestInviteIsAllowedOnlyForTheOwnerOfTheTeam(t *testing.T) {
	env := newTestEnvironment()
	team := env.createTeam(t, newTeamCaller(), "Furb FC")
	_, player := env.newPlayerUser()

	byStranger, strangerErr := env.useCase.Invite(context.Background(), newTeamCaller(), team.ID, player.ID)
	byAdmin, adminErr := env.useCase.Invite(context.Background(), newAdminCaller(), team.ID, player.ID)

	if !errors.Is(strangerErr, entities.ErrForbidden) || byStranger != nil {
		t.Errorf("a stranger must be forbidden: %v / %v", strangerErr, byStranger)
	}
	if !errors.Is(adminErr, entities.ErrForbidden) || byAdmin != nil {
		t.Errorf("only the owner invites, even the admin must be forbidden: %v / %v", adminErr, byAdmin)
	}
}

func TestInviteFailsForAnUnknownPlayer(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")

	invitation, err := env.useCase.Invite(context.Background(), owner, team.ID, uuid.New())

	if !errors.Is(err, entities.ErrNotFound) || invitation != nil {
		t.Errorf("want ErrNotFound and nil invitation, got %v / %v", err, invitation)
	}
}

func TestInviteRejectsAPlayerThatAlreadyHasATeam(t *testing.T) {
	env := newTestEnvironment()
	firstOwner := newTeamCaller()
	firstTeam := env.createTeam(t, firstOwner, "First")
	secondOwner := newTeamCaller()
	secondTeam := env.createTeam(t, secondOwner, "Second")
	playerCaller, player := env.newPlayerUser()
	invitation := env.invite(t, firstOwner, firstTeam, player)
	if _, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID); err != nil {
		t.Fatal(err)
	}

	second, err := env.useCase.Invite(context.Background(), secondOwner, secondTeam.ID, player.ID)

	if !errors.Is(err, entities.ErrPlayerAlreadyInTeam) || second != nil {
		t.Errorf("want ErrPlayerAlreadyInTeam and nil invitation, got %v / %v", err, second)
	}
}

func TestInviteRejectsADuplicatePendingInvitation(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()
	env.invite(t, owner, team, player)

	second, err := env.useCase.Invite(context.Background(), owner, team.ID, player.ID)

	if !errors.Is(err, entities.ErrInvitationAlreadyPending) || second != nil {
		t.Errorf("want ErrInvitationAlreadyPending and nil invitation, got %v / %v", err, second)
	}
}

func TestAcceptPutsThePlayerInTheTeamAndCancelsTheOtherInvitations(t *testing.T) {
	env := newTestEnvironment()
	firstOwner := newTeamCaller()
	firstTeam := env.createTeam(t, firstOwner, "First")
	secondOwner := newTeamCaller()
	secondTeam := env.createTeam(t, secondOwner, "Second")
	playerCaller, player := env.newPlayerUser()
	firstInvitation := env.invite(t, firstOwner, firstTeam, player)
	secondInvitation := env.invite(t, secondOwner, secondTeam, player)

	accepted, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, firstInvitation.ID)

	if err != nil || accepted.Status != entities.InvitationAccepted {
		t.Fatalf("accept failed: %+v / %v", accepted, err)
	}
	roster, _ := env.useCase.ListPlayers(context.Background(), firstTeam.ID)
	if len(roster) != 1 || roster[0].ID != player.ID {
		t.Errorf("the player must be in the roster, got %+v", roster)
	}
	other, _ := env.repo.FindInvitationByID(context.Background(), secondInvitation.ID)
	if other.Status != entities.InvitationCanceled {
		t.Errorf("the other pending invitation must be canceled, got %s", other.Status)
	}
}

func TestAcceptFailsForAnInvitationAddressedToAnotherPlayer(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()
	otherPlayerCaller, _ := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)

	accepted, err := env.useCase.AcceptInvitation(context.Background(), otherPlayerCaller, invitation.ID)

	if !errors.Is(err, entities.ErrForbidden) || accepted != nil {
		t.Errorf("want ErrForbidden and nil invitation, got %v / %v", err, accepted)
	}
}

func TestAcceptFailsWhenTheUserHasNoPlayerProfile(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)

	accepted, err := env.useCase.AcceptInvitation(context.Background(), newUserCaller(), invitation.ID)

	if !errors.Is(err, entities.ErrNotFound) || accepted != nil {
		t.Errorf("want ErrNotFound and nil invitation, got %v / %v", err, accepted)
	}
}

func TestAcceptTwiceFailsTheSecondTime(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	playerCaller, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)
	if _, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID); err != nil {
		t.Fatal(err)
	}

	again, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID)

	if !errors.Is(err, entities.ErrInvitationNotPending) || again != nil {
		t.Errorf("want ErrInvitationNotPending and nil invitation, got %v / %v", err, again)
	}
}

func TestDeclineMarksTheInvitationAndItCannotBeAcceptedAfterwards(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	playerCaller, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)

	declined, err := env.useCase.DeclineInvitation(context.Background(), playerCaller, invitation.ID)
	accepted, acceptErr := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID)

	if err != nil || declined.Status != entities.InvitationDeclined {
		t.Errorf("decline failed: %+v / %v", declined, err)
	}
	if !errors.Is(acceptErr, entities.ErrInvitationNotPending) || accepted != nil {
		t.Errorf("a declined invitation cannot be accepted: %v / %v", acceptErr, accepted)
	}
}

func TestCancelByTheOwnerChangesThePendingInvitation(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)

	canceled, err := env.useCase.CancelInvitation(context.Background(), owner, team.ID, invitation.ID)

	if err != nil || canceled.Status != entities.InvitationCanceled {
		t.Errorf("cancel failed: %+v / %v", canceled, err)
	}
}

func TestCancelFailsForStrangersAndForInvitationsOfAnotherTeam(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	otherOwner := newTeamCaller()
	otherTeam := env.createTeam(t, otherOwner, "Other FC")
	_, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)

	byStranger, strangerErr := env.useCase.CancelInvitation(context.Background(), otherOwner, team.ID, invitation.ID)
	fromAnotherTeam, anotherTeamErr := env.useCase.CancelInvitation(context.Background(), otherOwner, otherTeam.ID, invitation.ID)

	if !errors.Is(strangerErr, entities.ErrForbidden) || byStranger != nil {
		t.Errorf("a stranger must be forbidden: %v / %v", strangerErr, byStranger)
	}
	if !errors.Is(anotherTeamErr, entities.ErrNotFound) || fromAnotherTeam != nil {
		t.Errorf("an invitation of another team must look not found: %v / %v", anotherTeamErr, fromAnotherTeam)
	}
}

func TestRemovePlayerAndLeaveTeam(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	firstCaller, first := env.newPlayerUser()
	secondCaller, second := env.newPlayerUser()
	for _, pair := range []struct {
		caller *entities.Caller
		player entities.Player
	}{{firstCaller, first}, {secondCaller, second}} {
		invitation := env.invite(t, owner, team, pair.player)
		if _, err := env.useCase.AcceptInvitation(context.Background(), pair.caller, invitation.ID); err != nil {
			t.Fatal(err)
		}
	}

	removeByStrangerErr := env.useCase.RemovePlayer(context.Background(), newTeamCaller(), team.ID, first.ID)
	removeErr := env.useCase.RemovePlayer(context.Background(), owner, team.ID, first.ID)
	removeAgainErr := env.useCase.RemovePlayer(context.Background(), owner, team.ID, first.ID)
	leaveErr := env.useCase.LeaveTeam(context.Background(), secondCaller)
	leaveAgainErr := env.useCase.LeaveTeam(context.Background(), secondCaller)

	if !errors.Is(removeByStrangerErr, entities.ErrForbidden) {
		t.Errorf("a stranger cannot remove players: %v", removeByStrangerErr)
	}
	if removeErr != nil || leaveErr != nil {
		t.Errorf("owner removal and leaving must work: %v / %v", removeErr, leaveErr)
	}
	if !errors.Is(removeAgainErr, entities.ErrPlayerNotInTeam) || !errors.Is(leaveAgainErr, entities.ErrPlayerNotInTeam) {
		t.Errorf("repeating must say the player is not in the team: %v / %v", removeAgainErr, leaveAgainErr)
	}
	roster, _ := env.useCase.ListPlayers(context.Background(), team.ID)
	if len(roster) != 0 {
		t.Errorf("the roster must be empty, got %+v", roster)
	}
}

func TestLeaveTeamFailsForAUserWithoutAPlayerProfile(t *testing.T) {
	err := newTestEnvironment().useCase.LeaveTeam(context.Background(), newUserCaller())

	if !errors.Is(err, entities.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestGetMyTeamReturnsTheTeamTheyJoinedAndNotFoundOtherwise(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	playerCaller, player := env.newPlayerUser()
	freeAgentCaller, _ := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)
	if _, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID); err != nil {
		t.Fatal(err)
	}

	mine, err := env.useCase.GetMyTeam(context.Background(), playerCaller)
	none, noneErr := env.useCase.GetMyTeam(context.Background(), freeAgentCaller)
	noProfile, noProfileErr := env.useCase.GetMyTeam(context.Background(), newUserCaller())

	if err != nil || mine.ID != team.ID {
		t.Errorf("want the joined team, got %+v (err=%v)", mine, err)
	}
	if !errors.Is(noneErr, entities.ErrNotFound) || none != nil {
		t.Errorf("a player without a team must get ErrNotFound, got %v / %v", noneErr, none)
	}
	if !errors.Is(noProfileErr, entities.ErrNotFound) || noProfile != nil {
		t.Errorf("a user without a player profile must get ErrNotFound, got %v / %v", noProfileErr, noProfile)
	}
}

func TestListMyInvitationsReturnsOnlyTheInvitationsOfThePlayer(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	playerCaller, player := env.newPlayerUser()
	_, otherPlayer := env.newPlayerUser()
	env.invite(t, owner, team, player)
	env.invite(t, owner, team, otherPlayer)

	mine, err := env.useCase.ListMyInvitations(context.Background(), playerCaller)

	if err != nil || len(mine) != 1 || mine[0].PlayerID != player.ID {
		t.Errorf("want only my invitation, got %+v (err=%v)", mine, err)
	}
}

func TestListTeamInvitationsIsAllowedForTheOwnerAndTheAdminOnly(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	_, player := env.newPlayerUser()
	env.invite(t, owner, team, player)

	byOwner, ownerErr := env.useCase.ListTeamInvitations(context.Background(), owner, team.ID)
	byAdmin, adminErr := env.useCase.ListTeamInvitations(context.Background(), newAdminCaller(), team.ID)
	byStranger, strangerErr := env.useCase.ListTeamInvitations(context.Background(), newTeamCaller(), team.ID)

	if ownerErr != nil || len(byOwner) != 1 || adminErr != nil || len(byAdmin) != 1 {
		t.Errorf("owner and admin must see the invitation: %v / %v", ownerErr, adminErr)
	}
	if !errors.Is(strangerErr, entities.ErrForbidden) || byStranger != nil {
		t.Errorf("a stranger must be forbidden: %v / %v", strangerErr, byStranger)
	}
}

func TestPlayerBelongsToTeamOnlyForTheTeamTheyJoined(t *testing.T) {
	env := newTestEnvironment()
	owner := newTeamCaller()
	team := env.createTeam(t, owner, "Furb FC")
	otherTeam := env.createTeam(t, newTeamCaller(), "Other FC")
	playerCaller, player := env.newPlayerUser()
	invitation := env.invite(t, owner, team, player)
	if _, err := env.useCase.AcceptInvitation(context.Background(), playerCaller, invitation.ID); err != nil {
		t.Fatal(err)
	}

	inOwnTeam, ownErr := env.useCase.PlayerBelongsToTeam(context.Background(), team.ID, player.ID)
	inOtherTeam, otherErr := env.useCase.PlayerBelongsToTeam(context.Background(), otherTeam.ID, player.ID)

	if ownErr != nil || otherErr != nil || !inOwnTeam || inOtherTeam {
		t.Errorf("want true/false, got %v/%v (errs %v / %v)", inOwnTeam, inOtherTeam, ownErr, otherErr)
	}
}
