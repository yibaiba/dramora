package base62

import (
	"errors"
	"math"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrInvalidBase62 = errors.New("invalid base62 string")
)

// Encode converts a uint64 to a base62 string
func Encode(num uint64) string {
	if num == 0 {
		return "0"
	}

	var result strings.Builder
	for num > 0 {
		result.WriteByte(alphabet[num%62])
		num /= 62
	}

	// Reverse the result since we built it backwards
	s := result.String()
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Decode converts a base62 string to a uint64
func Decode(s string) (uint64, error) {
	if s == "" {
		return 0, ErrInvalidBase62
	}

	var result uint64
	for _, c := range s {
		digit := strings.IndexRune(alphabet, c)
		if digit == -1 {
			return 0, ErrInvalidBase62
		}

		// Check for overflow before multiplying
		if result > math.MaxUint64/62 {
			return 0, ErrInvalidBase62
		}
		result = result*62 + uint64(digit)
	}

	return result, nil
}
