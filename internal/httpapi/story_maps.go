package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

func (api *api) getStoryMap(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	storyMap, err := api.productionService.GetStoryMap(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"story_map": storyMapDTO(storyMap)})
}

func (api *api) seedStoryMap(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	storyMap, err := api.productionService.SeedStoryMap(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"story_map": storyMapDTO(storyMap)})
}

type saveCharacterBibleRequest struct {
	CharacterBible characterBibleRequest `json:"character_bible"`
}

type characterBibleRequest struct {
	Anchor          string                                `json:"anchor"`
	Palette         characterBiblePaletteRequest          `json:"palette"`
	Expressions     []string                              `json:"expressions"`
	ReferenceAngles []string                              `json:"reference_angles"`
	ReferenceAssets []characterBibleReferenceAssetRequest `json:"reference_assets"`
	Wardrobe        string                                `json:"wardrobe"`
	Notes           string                                `json:"notes"`
}

type characterBiblePaletteRequest struct {
	Skin    string `json:"skin"`
	Hair    string `json:"hair"`
	Accent  string `json:"accent"`
	Eyes    string `json:"eyes"`
	Costume string `json:"costume"`
}

type characterBibleReferenceAssetRequest struct {
	Angle   string `json:"angle"`
	AssetID string `json:"asset_id"`
}

func (api *api) saveCharacterBible(w http.ResponseWriter, r *http.Request) {
	characterID := chi.URLParam(r, "characterId")

	var request saveCharacterBibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	character, err := api.productionService.SaveCharacterBible(r.Context(), characterID, service.SaveCharacterBibleInput{
		CharacterBible: request.toDomain(),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"story_map_item": characterDTO(character)})
}

func (request saveCharacterBibleRequest) toDomain() domain.CharacterBible {
	return domain.CharacterBible{
		Anchor: request.CharacterBible.Anchor,
		Palette: domain.CharacterBiblePalette{
			Skin:    request.CharacterBible.Palette.Skin,
			Hair:    request.CharacterBible.Palette.Hair,
			Accent:  request.CharacterBible.Palette.Accent,
			Eyes:    request.CharacterBible.Palette.Eyes,
			Costume: request.CharacterBible.Palette.Costume,
		},
		Expressions:     request.CharacterBible.Expressions,
		ReferenceAngles: request.CharacterBible.ReferenceAngles,
		ReferenceAssets: request.CharacterBible.toDomainReferenceAssets(),
		Wardrobe:        request.CharacterBible.Wardrobe,
		Notes:           request.CharacterBible.Notes,
	}
}

func (request characterBibleRequest) toDomainReferenceAssets() []domain.CharacterBibleReferenceAsset {
	references := make([]domain.CharacterBibleReferenceAsset, 0, len(request.ReferenceAssets))
	for _, item := range request.ReferenceAssets {
		references = append(references, domain.CharacterBibleReferenceAsset{
			Angle:   item.Angle,
			AssetID: item.AssetID,
		})
	}
	return references
}
