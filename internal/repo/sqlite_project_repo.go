package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yibaiba/dramora/internal/domain"
)

type SQLiteProjectRepository struct {
	db *sql.DB
}

func NewSQLiteProjectRepository(db *sql.DB) *SQLiteProjectRepository {
	return &SQLiteProjectRepository{db: db}
}

func (r *SQLiteProjectRepository) ListProjects(ctx context.Context, organizationID string) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListProjectsSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	return sqliteScanProjects(rows)
}

func (r *SQLiteProjectRepository) CreateProject(ctx context.Context, params CreateProjectParams) (domain.Project, error) {
	_, err := r.db.ExecContext(ctx, sqliteCreateProjectSQL,
		params.ID, params.OrganizationID, params.Name, params.Description, params.Status,
	)
	if err != nil {
		if isSQLiteFKViolation(err) {
			return domain.Project{}, domain.ErrNotFound
		}
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}
	return r.GetProject(ctx, params.OrganizationID, params.ID)
}

func (r *SQLiteProjectRepository) GetProject(ctx context.Context, organizationID string, projectID string) (domain.Project, error) {
	row := r.db.QueryRowContext(ctx, sqliteGetProjectSQL, projectID, organizationID)
	project, err := scanSQLiteProject(row)
	if err == sql.ErrNoRows {
		return domain.Project{}, domain.ErrNotFound
	}
	return project, err
}

func (r *SQLiteProjectRepository) LookupProjectByID(ctx context.Context, projectID string) (domain.Project, error) {
	row := r.db.QueryRowContext(ctx, sqliteLookupProjectByIDSQL, projectID)
	project, err := scanSQLiteProject(row)
	if err == sql.ErrNoRows {
		return domain.Project{}, domain.ErrNotFound
	}
	return project, err
}

func (r *SQLiteProjectRepository) ListEpisodes(ctx context.Context, projectID string) ([]domain.Episode, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListEpisodesSQL, projectID)
	if err != nil {
		return nil, fmt.Errorf("list episodes: %w", err)
	}
	defer rows.Close()

	return sqliteScanEpisodes(rows)
}

func (r *SQLiteProjectRepository) CreateEpisode(ctx context.Context, params CreateEpisodeParams) (domain.Episode, error) {
	_, err := r.db.ExecContext(ctx, sqliteCreateEpisodeSQL,
		params.ID, params.ProjectID, params.Number, params.Title, params.Status,
	)
	if err != nil {
		if isSQLiteFKViolation(err) {
			return domain.Episode{}, domain.ErrNotFound
		}
		if isSQLiteUniqueViolation(err) {
			return domain.Episode{}, domain.ErrInvalidInput
		}
		return domain.Episode{}, fmt.Errorf("create episode: %w", err)
	}
	return r.GetEpisode(ctx, params.ID)
}

func (r *SQLiteProjectRepository) GetEpisode(ctx context.Context, episodeID string) (domain.Episode, error) {
	row := r.db.QueryRowContext(ctx, sqliteGetEpisodeSQL, episodeID)
	episode, err := scanSQLiteEpisode(row)
	if err == sql.ErrNoRows {
		return domain.Episode{}, domain.ErrNotFound
	}
	return episode, err
}

func sqliteScanProjects(rows *sql.Rows) ([]domain.Project, error) {
	projects := make([]domain.Project, 0)
	for rows.Next() {
		project, err := scanSQLiteProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func sqliteScanEpisodes(rows *sql.Rows) ([]domain.Episode, error) {
	episodes := make([]domain.Episode, 0)
	for rows.Next() {
		episode, err := scanSQLiteEpisode(rows)
		if err != nil {
			return nil, err
		}
		episodes = append(episodes, episode)
	}
	return episodes, rows.Err()
}

func scanSQLiteProject(row rowScanner) (domain.Project, error) {
	var (
		project   domain.Project
		createdAt string
		updatedAt string
	)
	if err := row.Scan(
		&project.ID,
		&project.OrganizationID,
		&project.Name,
		&project.Description,
		&project.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Project{}, fmt.Errorf("scan project: %w", err)
	}
	var err error
	if project.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Project{}, err
	}
	if project.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func scanSQLiteEpisode(row rowScanner) (domain.Episode, error) {
	var (
		episode   domain.Episode
		createdAt string
		updatedAt string
	)
	if err := row.Scan(
		&episode.ID,
		&episode.ProjectID,
		&episode.Number,
		&episode.Title,
		&episode.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Episode{}, fmt.Errorf("scan episode: %w", err)
	}
	var err error
	if episode.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Episode{}, err
	}
	if episode.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Episode{}, err
	}
	return episode, nil
}
