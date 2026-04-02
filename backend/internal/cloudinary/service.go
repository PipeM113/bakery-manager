package cloudinary

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"

	cld "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Service struct {
	client *cld.Cloudinary
}

func NewService() (*Service, error) {
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL no está configurada")
	}
	client, err := cld.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, fmt.Errorf("inicializar cloudinary: %w", err)
	}
	return &Service{client: client}, nil
}

func (s *Service) UploadRecipePhoto(ctx context.Context, fileHeader *multipart.FileHeader, recipeID string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("abrir archivo: %w", err)
	}
	defer file.Close()

	overwrite := true
	resp, err := s.client.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:    fmt.Sprintf("bakery/recipes/%s", recipeID),
		PublicID:  "photo",
		Overwrite: &overwrite,
	})
	if err != nil {
		return "", fmt.Errorf("subir a cloudinary: %w", err)
	}
	return resp.SecureURL, nil
}
