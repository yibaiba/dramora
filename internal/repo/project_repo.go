package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

type CreateProjectParams struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	Status         domain.ProjectStatus
}

type CreateEpisodeParams struct {
	ID        string
	ProjectID string
	Number    int
	Title     string
	Status    domain.EpisodeStatus
}

type ProjectRepository interface {
	ListProjects(ctx context.Context, organizationID string) ([]domain.Project, error)
	CreateProject(ctx context.Context, params CreateProjectParams) (domain.Project, error)
	GetProject(ctx context.Context, organizationID string, projectID string) (domain.Project, error)
	ListEpisodes(ctx context.Context, projectID string) ([]domain.Episode, error)
	CreateEpisode(ctx context.Context, params CreateEpisodeParams) (domain.Episode, error)
	GetEpisode(ctx context.Context, episodeID string) (domain.Episode, error)
}

type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

func (r *PostgresProjectRepository) ListProjects(ctx context.Context, organizationID string) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, listProjectsSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

func (r *PostgresProjectRepository) CreateProject(
	ctx context.Context,
	params CreateProjectParams,
) (domain.Project, error) {
	row := r.pool.QueryRow(ctx, createProjectSQL,
		params.ID,
		params.OrganizationID,
		params.Name,
		params.Description,
		params.Status,
	)
	project, err := scanProject(row)
	if isForeignKeyViolation(err) {
		return domain.Project{}, domain.ErrNotFound
	}
	return project, err
}

func (r *PostgresProjectRepository) GetProject(
	ctx context.Context,
	organizationID string,
	projectID string,
) (domain.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, getProjectSQL, projectID, organizationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}
	return project, err
}

func (r *PostgresProjectRepository) ListEpisodes(ctx context.Context, projectID string) ([]domain.Episode, error) {
	rows, err := r.pool.Query(ctx, listEpisodesSQL, projectID)
	if err != nil {
		return nil, fmt.Errorf("list episodes: %w", err)
	}
	defer rows.Close()

	return scanEpisodes(rows)
}

func (r *PostgresProjectRepository) CreateEpisode(
	ctx context.Context,
	params CreateEpisodeParams,
) (domain.Episode, error) {
	episode, err := scanEpisode(r.pool.QueryRow(ctx, createEpisodeSQL,
		params.ID,
		params.ProjectID,
		params.Number,
		params.Title,
		params.Status,
	))
	if isForeignKeyViolation(err) {
		return domain.Episode{}, domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.Episode{}, domain.ErrInvalidInput
	}
	return episode, err
}

func (r *PostgresProjectRepository) GetEpisode(ctx context.Context, episodeID string) (domain.Episode, error) {
	episode, err := scanEpisode(r.pool.QueryRow(ctx, getEpisodeSQL, episodeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Episode{}, domain.ErrNotFound
	}
	return episode, err
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
