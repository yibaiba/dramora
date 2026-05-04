package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShortVideoTemplateValidate(t *testing.T) {
	tests := []struct {
		name      string
		template  *ShortVideoTemplate
		wantError bool
	}{
		{
			name: "valid template",
			template: &ShortVideoTemplate{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Name:           "Product Focus",
				Description:    "Template for highlighting product features",
				Category:       "product-focus",
				Config:         json.RawMessage(`{"duration": 30}`),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			wantError: false,
		},
		{
			name: "missing name",
			template: &ShortVideoTemplate{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Name:           "",
				Category:       "product-focus",
				Config:         json.RawMessage(`{"duration": 30}`),
			},
			wantError: true,
		},
		{
			name: "missing category",
			template: &ShortVideoTemplate{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Name:           "Product Focus",
				Category:       "",
				Config:         json.RawMessage(`{"duration": 30}`),
			},
			wantError: true,
		},
		{
			name: "empty config",
			template: &ShortVideoTemplate{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Name:           "Product Focus",
				Category:       "product-focus",
				Config:         json.RawMessage(``),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestShortVideoValidate(t *testing.T) {
	tests := []struct {
		name      string
		video     *ShortVideo
		wantError bool
	}{
		{
			name: "valid short video",
			video: &ShortVideo{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				TemplateID:     uuid.New(),
				Parameters:     json.RawMessage(`{"productName": "iPhone 15"}`),
				Status:         ShortVideoStatusPending,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			wantError: false,
		},
		{
			name: "missing template id",
			video: &ShortVideo{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				TemplateID:     uuid.Nil,
				Parameters:     json.RawMessage(`{"productName": "iPhone 15"}`),
				Status:         ShortVideoStatusPending,
			},
			wantError: true,
		},
		{
			name: "empty parameters",
			video: &ShortVideo{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				TemplateID:     uuid.New(),
				Parameters:     json.RawMessage(``),
				Status:         ShortVideoStatusPending,
			},
			wantError: true,
		},
		{
			name: "empty status",
			video: &ShortVideo{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				TemplateID:     uuid.New(),
				Parameters:     json.RawMessage(`{"productName": "iPhone 15"}`),
				Status:         "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.video.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		current string
		next    string
		want    bool
	}{
		// Valid transitions
		{ShortVideoStatusPending, ShortVideoStatusGenerating, true},
		{ShortVideoStatusGenerating, ShortVideoStatusCompleted, true},
		{ShortVideoStatusGenerating, ShortVideoStatusFailed, true},

		// Invalid transitions
		{ShortVideoStatusPending, ShortVideoStatusCompleted, false},
		{ShortVideoStatusPending, ShortVideoStatusFailed, false},
		{ShortVideoStatusCompleted, ShortVideoStatusGenerating, false},
		{ShortVideoStatusFailed, ShortVideoStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.next, func(t *testing.T) {
			if got := CanTransitionTo(tt.current, tt.next); got != tt.want {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.current, tt.next, got, tt.want)
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{ShortVideoStatusCompleted, true},
		{ShortVideoStatusFailed, true},
		{ShortVideoStatusPending, false},
		{ShortVideoStatusGenerating, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := IsTerminalStatus(tt.status); got != tt.want {
				t.Errorf("IsTerminalStatus(%s) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
