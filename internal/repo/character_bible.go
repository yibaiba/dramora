package repo

import (
	"bytes"
	"encoding/json"

	"github.com/yibaiba/dramora/internal/domain"
)

func encodeCharacterBible(bible domain.CharacterBible) (string, error) {
	payload, err := json.Marshal(bible)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeCharacterBible(payload []byte) (*domain.CharacterBible, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("{}")) || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var bible domain.CharacterBible
	if err := json.Unmarshal(trimmed, &bible); err != nil {
		return nil, err
	}
	if bible.Expressions == nil {
		bible.Expressions = []string{}
	}
	if bible.ReferenceAngles == nil {
		bible.ReferenceAngles = []string{}
	}
	if bible.ReferenceAssets == nil {
		bible.ReferenceAssets = []domain.CharacterBibleReferenceAsset{}
	}
	return &bible, nil
}
