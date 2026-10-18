// Package storage uploads recipe photos to a public bucket of Supabase Storage.
//
// It talks to the Storage REST API with the project's secret key, which has full access to
// the project: it only ever travels in request headers, and no error carries it.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Config holds the project origin (https://<ref>.supabase.co), the secret key and the bucket.
type Config struct {
	URL       string
	SecretKey string
	Bucket    string
}

// Client uploads photos. Build it with New.
type Client struct {
	base, key, bucket string
	http              *http.Client
}

// safeSegment limits what may end up inside the object path.
var safeSegment = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

var extensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// New builds a client; with a nil httpClient it uses one with a 20 second timeout.
func New(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		base:   strings.TrimRight(cfg.URL, "/"),
		key:    cfg.SecretKey,
		bucket: cfg.Bucket,
		http:   httpClient,
	}
}

// UploadRecipePhoto stores data under recipes/<recipeID>/<random name><extension> and returns
// the public URL. Every upload gets a new name: overwriting would leave the old image in the
// CDN cache. The extension matters: the PDF code reads it to pick the image type.
func (c *Client) UploadRecipePhoto(ctx context.Context, recipeID, contentType string, data []byte) (string, error) {
	if !safeSegment.MatchString(recipeID) {
		return "", fmt.Errorf("id de receta inválido")
	}
	ext, ok := extensions[contentType]
	if !ok {
		return "", fmt.Errorf("tipo de imagen no admitido")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("la foto está vacía")
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("no se pudo generar el nombre de la foto")
	}
	path := "recipes/" + recipeID + "/" + hex.EncodeToString(random) + ext

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/storage/v1/object/"+c.bucket+"/"+path, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("no se pudo armar la solicitud de subida")
	}
	req.Header.Set("apikey", c.key)
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		// The error of net/http names the URL, never the headers: the key is not in it.
		return "", fmt.Errorf("subir la foto: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// The body is not repeated: it is the server's text, not ours to forward.
		return "", fmt.Errorf("el almacenamiento respondió %d", resp.StatusCode)
	}
	return c.base + "/storage/v1/object/public/" + c.bucket + "/" + path, nil
}
