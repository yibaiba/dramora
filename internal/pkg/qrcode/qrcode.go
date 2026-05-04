package qrcode

import (
	"encoding/base64"

	qr "github.com/skip2/go-qrcode"
)

// GenerateQRCode generates a QR code as PNG bytes
func GenerateQRCode(code string) ([]byte, error) {
	// Create QR code with 200x200 size (standard for emails)
	qr, err := qr.New(code, qr.Medium)
	if err != nil {
		return nil, err
	}

	// Generate PNG bytes
	pngBytes, err := qr.PNG(200)
	if err != nil {
		return nil, err
	}

	return pngBytes, nil
}

// GenerateQRCodeBase64 generates a QR code and encodes it to base64
func GenerateQRCodeBase64(code string) (string, error) {
	pngBytes, err := GenerateQRCode(code)
	if err != nil {
		return "", err
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(pngBytes), nil
}

// GenerateQRCodeDataURL generates a QR code as a data URL (for direct HTML embedding)
func GenerateQRCodeDataURL(code string) (string, error) {
	base64Str, err := GenerateQRCodeBase64(code)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64Str, nil
}
