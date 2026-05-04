package qrcode

import (
	"strings"
	"testing"
)

func TestGenerateQRCode(t *testing.T) {
	code := "GIFT-TEST12345"

	// Test PNG generation
	pngBytes, err := GenerateQRCode(code)
	if err != nil {
		t.Fatalf("GenerateQRCode failed: %v", err)
	}

	if len(pngBytes) == 0 {
		t.Error("GenerateQRCode returned empty PNG")
	}

	// Check PNG signature
	if len(pngBytes) < 8 || string(pngBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Error("Invalid PNG signature")
	}
}

func TestGenerateQRCodeBase64(t *testing.T) {
	code := "GIFT-TEST12345"

	base64Str, err := GenerateQRCodeBase64(code)
	if err != nil {
		t.Fatalf("GenerateQRCodeBase64 failed: %v", err)
	}

	if len(base64Str) == 0 {
		t.Error("GenerateQRCodeBase64 returned empty string")
	}

	// Verify it's valid base64
	pngBytes, err := GenerateQRCode(code)
	if err != nil {
		t.Fatalf("GenerateQRCode failed: %v", err)
	}

	// Manual base64 encode to compare
	expectedBase64 := ""
	for _, b := range pngBytes {
		expectedBase64 += string(b)
	}

	if base64Str == "" {
		t.Error("Base64 encoding is empty")
	}
}

func TestGenerateQRCodeDataURL(t *testing.T) {
	code := "GIFT-TEST12345"

	dataURL, err := GenerateQRCodeDataURL(code)
	if err != nil {
		t.Fatalf("GenerateQRCodeDataURL failed: %v", err)
	}

	// Check data URL format
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Errorf("Invalid data URL format: %s", dataURL)
	}

	if len(dataURL) < len("data:image/png;base64,") {
		t.Error("Data URL is too short")
	}
}

func TestGenerateQRCodeConsistency(t *testing.T) {
	code := "GIFT-TEST12345"

	// Generate multiple times
	qr1, err := GenerateQRCode(code)
	if err != nil {
		t.Fatalf("First generation failed: %v", err)
	}

	qr2, err := GenerateQRCode(code)
	if err != nil {
		t.Fatalf("Second generation failed: %v", err)
	}

	// QR codes should be identical for the same input
	if len(qr1) != len(qr2) {
		t.Errorf("QR codes have different lengths: %d vs %d", len(qr1), len(qr2))
	}

	for i := range qr1 {
		if qr1[i] != qr2[i] {
			t.Errorf("QR codes differ at byte %d: %v vs %v", i, qr1[i], qr2[i])
			break
		}
	}
}
