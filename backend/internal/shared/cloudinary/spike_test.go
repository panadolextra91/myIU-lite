package cloudinary_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/cloudinary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudinarySpike(t *testing.T) {
	// Live external-service spike: hits the real Cloudinary API over the network.
	// It retired the Wave-1 A1 risk during development and is OPT-IN ONLY — it must
	// not run in CI (CI sets a placeholder CLOUDINARY_URL with no real account, and
	// CI shouldn't depend on an external service/secrets). Run it on demand with
	// real credentials:
	//   RUN_CLOUDINARY_SPIKE=1 CLOUDINARY_URL=cloudinary://<key>:<secret>@<cloud> \
	//     go test ./internal/shared/cloudinary/ -run TestCloudinarySpike
	if os.Getenv("RUN_CLOUDINARY_SPIKE") != "1" {
		t.Skip("opt-in: set RUN_CLOUDINARY_SPIKE=1 (with a real CLOUDINARY_URL) to run the live Cloudinary spike")
	}
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		t.Skip("CLOUDINARY_URL is not set")
	}

	cld, err := cloudinary.New(cloudinaryURL)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Upload a dummy PDF
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<<\n/Title (Dummy PDF)\n>>\nendobj\ntrailer\n<<\n/Root 1 0 R\n>>\n%%EOF")
	reader := bytes.NewReader(pdfBytes)

	publicID, format, err := cld.Upload(ctx, reader, "spike_tests")
	require.NoError(t, err)
	require.NotEmpty(t, publicID)

	// 2. Get signed download URL
	downloadURL, err := cld.SignedDownloadURL(publicID, format)
	require.NoError(t, err)
	require.NotEmpty(t, downloadURL)

	// 3. HTTP GET to verify
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, pdfBytes, body)
}
