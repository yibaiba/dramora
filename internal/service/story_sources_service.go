package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

const maxStorySourceChars = 20000

type CreateStorySourceInput struct {
	SourceType  string
	Title       string
	ContentText string
	Language    string
}

func (s *ProductionService) CreateStorySource(
	ctx context.Context,
	episode domain.Episode,
	input CreateStorySourceInput,
) (domain.StorySource, error) {
	content := strings.TrimSpace(input.ContentText)
	if content == "" {
		return domain.StorySource{}, fmt.Errorf("%w: story source content is required", domain.ErrInvalidInput)
	}
	if len([]rune(content)) > maxStorySourceChars {
		return domain.StorySource{}, fmt.Errorf("%w: story source content exceeds %d characters", domain.ErrInvalidInput, maxStorySourceChars)
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.StorySource{}, err
	}
	return s.production.CreateStorySource(ctx, repo.CreateStorySourceParams{
		ID: id, ProjectID: episode.ProjectID, EpisodeID: episode.ID,
		SourceType:  normalizedStorySourceType(input.SourceType),
		Title:       strings.TrimSpace(input.Title),
		ContentText: content,
		Language:    normalizedStoryLanguage(input.Language),
	})
}

func (s *ProductionService) ListStorySources(ctx context.Context, episodeID string) ([]domain.StorySource, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.ListStorySources(ctx, episodeID)
}

func (s *ProductionService) LatestStorySource(ctx context.Context, episodeID string) (domain.StorySource, error) {
	if strings.TrimSpace(episodeID) == "" {
		return domain.StorySource{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.LatestStorySource(ctx, episodeID)
}

func normalizedStorySourceType(value string) string {
	switch strings.TrimSpace(value) {
	case "idea", "outline", "novel", "script", "file", "url":
		return strings.TrimSpace(value)
	default:
		return "novel"
	}
}

func normalizedStoryLanguage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "zh-CN"
	}
	return value
}
