package base62

import (
	"math"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"ten", 10, "A"},
		{"sixtyone", 61, "z"},
		{"sixtytwo", 62, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"one", "1", 1, false},
		{"ten", "A", 10, false},
		{"sixtyone", "z", 61, false},
		{"sixtytwo", "10", 62, false},
		{"invalid", "!!!!", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil && result != tt.want {
				t.Errorf("Decode(%q) = %d, want %d", tt.input, result, tt.want)
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	tests := []uint64{
		0, 1, 10, 61, 62, 100, 1000, 10000, 100000, 1000000,
		math.MaxInt32, math.MaxInt64, math.MaxUint64 - 1,
	}

	for _, num := range tests {
		t.Run("roundtrip", func(t *testing.T) {
			encoded := Encode(num)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Errorf("Decode(Encode(%d)) failed: %v", num, err)
				return
			}
			if decoded != num {
				t.Errorf("Roundtrip(%d) = %d, mismatch", num, decoded)
			}
		})
	}
}

func TestDecodeOverflow(t *testing.T) {
	// Create a string that would overflow when decoded
	// "z" * 20 should overflow since each character adds significant weight
	longStr := "zzzzzzzzzzzzzzzzzzzz"
	_, err := Decode(longStr)
	if err == nil {
		t.Errorf("Decode(%q) should overflow, but got no error", longStr)
	}
}
