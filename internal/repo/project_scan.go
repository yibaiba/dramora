package repo

import (
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	rowScanner
	Next() bool
	Err() error
}

func scanProjects(rows pgx.Rows) ([]domain.Project, error) {
	projects := make([]domain.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan projects: %w", err)
	}
	return projects, nil
}

func scanProject(row rowScanner) (domain.Project, error) {
	var project domain.Project
	if err := row.Scan(
		&project.ID,
		&project.OrganizationID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	); err != nil {
		return domain.Project{}, fmt.Errorf("scan project: %w", err)
	}
	return project, nil
}

func scanEpisodes(rows pgx.Rows) ([]domain.Episode, error) {
	episodes := make([]domain.Episode, 0)
	for rows.Next() {
		episode, err := scanEpisode(rows)
		if err != nil {
			return nil, err
		}
		episodes = append(episodes, episode)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan episodes: %w", err)
	}
	return episodes, nil
}

func scanEpisode(row rowScanner) (domain.Episode, error) {
	var episode domain.Episode
	if err := row.Scan(
		&episode.ID,
		&episode.ProjectID,
		&episode.Number,
		&episode.Title,
		&episode.Status,
		&episode.CreatedAt,
		&episode.UpdatedAt,
	); err != nil {
		return domain.Episode{}, fmt.Errorf("scan episode: %w", err)
	}
	return episode, nil
}
