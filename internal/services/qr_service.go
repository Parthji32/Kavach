package services

import (
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRCode generates a QR code image encoding the given URL
// and returns it as a base64 data URI (PNG format)
func GenerateQRCode(triggerURL string) (string, error) {
	// Generate QR code as PNG bytes at medium recovery level, 256x256 pixels
	png, err := qrcode.Encode(triggerURL, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("qr code generation failed: %w", err)
	}

	// Convert to base64 data URI
	b64 := base64.StdEncoding.EncodeToString(png)
	dataURI := fmt.Sprintf("data:image/png;base64,%s", b64)

	return dataURI, nil
}
