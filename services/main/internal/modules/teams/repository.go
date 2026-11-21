package teams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/util"
)

const (
	teamColumns       = "id, owner_id, name, city, created_at, updated_at"
	invitationColumns = "i.id, i.team_id, t.name, i.player_id, i.status, i.created_at, i.updated_at"
	invitationSource  = "FROM team_invitations i JOIN teams t ON t.id = i.team_id"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) InsertTeam(ctx context.Context, team *entities.Team) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO teams (id, owner_id, name, city, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		team.ID, team.OwnerID, team.Name, team.City, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrTeamNameTaken
		}
		return util.SendMessageWithError("repository error inserting team", err)
	}
	return nil
}

func (r *postgresRepository) FindTeamByID(ctx context.Context, id uuid.UUID) (*entities.Team, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+teamColumns+" FROM teams WHERE id = $1", id)
	return scanTeam(row)
}

func (r *postgresRepository) FindTeamsByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+teamColumns+" FROM teams WHERE id = ANY($1) ORDER BY name, id", ids)
	if err != nil {
		return nil, util.SendMessageWithError("repository error finding teams by ids", err)
	}
	return collectTeams(rows)
}

func (r *postgresRepository) ListTeams(ctx context.Context) ([]entities.Team, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+teamColumns+" FROM teams ORDER BY name, id")
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing teams", err)
	}
	return collectTeams(rows)
}

func (r *postgresRepository) ListTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]entities.Team, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+teamColumns+" FROM teams WHERE owner_id = $1 ORDER BY name, id", ownerID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing teams by owner", err)
	}
	return collectTeams(rows)
}

func (r *postgresRepository) UpdateTeam(ctx context.Context, team *entities.Team) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE teams SET name = $2, city = $3, updated_at = $4 WHERE id = $1",
		team.ID, team.Name, team.City, team.UpdatedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrTeamNameTaken
		}
		return util.SendMessageWithError("repository error updating team", err)
	}
	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: team", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM teams WHERE id = $1", id)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return entities.ErrTeamInUse
		}
		return util.SendMessageWithError("repository error deleting team", err)
	}
	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: team", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) ListMemberPlayerIDs(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, "SELECT player_id FROM team_members WHERE team_id = $1 ORDER BY joined_at", teamID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing team members", err)
	}
	defer rows.Close()

	playerIDs := []uuid.UUID{}
	for rows.Next() {
		var playerID uuid.UUID
		err := rows.Scan(&playerID)
		if err != nil {
			return nil, util.SendMessageWithError("repository error scanning team member", err)
		}
		playerIDs = append(playerIDs, playerID)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading team members", err)
	}
	return playerIDs, nil
}

func (r *postgresRepository) PlayerIsInTeam(ctx context.Context, playerID uuid.UUID) (bool, error) {
	var isInTeam bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM team_members WHERE player_id = $1)", playerID).Scan(&isInTeam)
	if err != nil {
		return false, util.SendMessageWithError("repository error checking player membership", err)
	}
	return isInTeam, nil
}

func (r *postgresRepository) PlayerIsMemberOfTeam(ctx context.Context, teamID, playerID uuid.UUID) (bool, error) {
	var isMember bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM team_members WHERE team_id = $1 AND player_id = $2)", teamID, playerID).Scan(&isMember)
	if err != nil {
		return false, util.SendMessageWithError("repository error checking membership of a team", err)
	}
	return isMember, nil
}

func (r *postgresRepository) DeleteMembership(ctx context.Context, teamID, playerID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM team_members WHERE team_id = $1 AND player_id = $2", teamID, playerID)
	if err != nil {
		return util.SendMessageWithError("repository error deleting membership", err)
	}
	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return entities.ErrPlayerNotInTeam
	}
	return nil
}

func (r *postgresRepository) DeleteMembershipByPlayer(ctx context.Context, playerID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM team_members WHERE player_id = $1", playerID)
	if err != nil {
		return util.SendMessageWithError("repository error deleting membership by player", err)
	}
	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return entities.ErrPlayerNotInTeam
	}
	return nil
}

func (r *postgresRepository) InsertInvitation(ctx context.Context, invitation *entities.Invitation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO team_invitations (id, team_id, player_id, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		invitation.ID, invitation.TeamID, invitation.PlayerID, string(invitation.Status), invitation.CreatedAt, invitation.UpdatedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrInvitationAlreadyPending
		}
		return util.SendMessageWithError("repository error inserting invitation", err)
	}
	return nil
}

func (r *postgresRepository) FindInvitationByID(ctx context.Context, id uuid.UUID) (*entities.Invitation, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+invitationColumns+" "+invitationSource+" WHERE i.id = $1", id)
	return scanInvitation(row)
}

func (r *postgresRepository) ListInvitationsByTeam(ctx context.Context, teamID uuid.UUID) ([]entities.Invitation, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+invitationColumns+" "+invitationSource+" WHERE i.team_id = $1 ORDER BY i.created_at DESC", teamID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing invitations by team", err)
	}
	return collectInvitations(rows)
}

func (r *postgresRepository) ListInvitationsByPlayer(ctx context.Context, playerID uuid.UUID) ([]entities.Invitation, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+invitationColumns+" "+invitationSource+" WHERE i.player_id = $1 ORDER BY i.created_at DESC", playerID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing invitations by player", err)
	}
	return collectInvitations(rows)
}

func (r *postgresRepository) UpdateInvitationStatus(ctx context.Context, id uuid.UUID, from, to entities.InvitationStatus, updatedAt time.Time) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE team_invitations SET status = $3, updated_at = $4 WHERE id = $1 AND status = $2",
		id, string(from), string(to), updatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error updating invitation status", err)
	}
	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return entities.ErrInvitationNotPending
	}
	return nil
}

func (r *postgresRepository) AcceptInvitation(ctx context.Context, invitation *entities.Invitation, acceptedAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error starting accept invitation transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := tx.Exec(ctx,
		"UPDATE team_invitations SET status = 'ACCEPTED', updated_at = $2 WHERE id = $1 AND status = 'PENDING'",
		invitation.ID, acceptedAt)
	if err != nil {
		return util.SendMessageWithError("repository error marking invitation as accepted", err)
	}
	invitationWasNotPending := result.RowsAffected() == 0
	if invitationWasNotPending {
		return entities.ErrInvitationNotPending
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO team_members (player_id, team_id, joined_at) VALUES ($1, $2, $3)",
		invitation.PlayerID, invitation.TeamID, acceptedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrPlayerAlreadyInTeam
		}
		return util.SendMessageWithError("repository error adding player to team", err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE team_invitations SET status = 'CANCELED', updated_at = $2 WHERE player_id = $1 AND status = 'PENDING'",
		invitation.PlayerID, acceptedAt)
	if err != nil {
		return util.SendMessageWithError("repository error canceling the other invitations", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error committing accept invitation", err)
	}
	return nil
}

func collectTeams(rows pgx.Rows) ([]entities.Team, error) {
	defer rows.Close()

	allTeams := []entities.Team{}
	for rows.Next() {
		team, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		allTeams = append(allTeams, *team)
	}

	err := rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading teams", err)
	}
	return allTeams, nil
}

func collectInvitations(rows pgx.Rows) ([]entities.Invitation, error) {
	defer rows.Close()

	allInvitations := []entities.Invitation{}
	for rows.Next() {
		invitation, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		allInvitations = append(allInvitations, *invitation)
	}

	err := rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading invitations", err)
	}
	return allInvitations, nil
}

func scanTeam(row pgx.Row) (*entities.Team, error) {
	var team entities.Team

	err := row.Scan(&team.ID, &team.OwnerID, &team.Name, &team.City, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: team", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning team", err)
	}

	team.CreatedAt = team.CreatedAt.UTC()
	team.UpdatedAt = team.UpdatedAt.UTC()
	return &team, nil
}

func scanInvitation(row pgx.Row) (*entities.Invitation, error) {
	var invitation entities.Invitation
	var status string

	err := row.Scan(&invitation.ID, &invitation.TeamID, &invitation.TeamName, &invitation.PlayerID, &status, &invitation.CreatedAt, &invitation.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: invitation", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning invitation", err)
	}

	invitation.Status = entities.InvitationStatus(status)
	invitation.CreatedAt = invitation.CreatedAt.UTC()
	invitation.UpdatedAt = invitation.UpdatedAt.UTC()
	return &invitation, nil
}
