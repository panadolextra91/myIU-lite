package cloudinary

import (
	"context"
	"io"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Client struct {
	cld *cloudinary.Cloudinary
}

func New(cloudinaryURL string) (*Client, error) {
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, err
	}
	return &Client{cld: cld}, nil
}

func (c *Client) Upload(ctx context.Context, reader io.Reader, folder string) (string, string, error) {
	res, err := c.cld.Upload.Upload(ctx, reader, uploader.UploadParams{
		ResourceType: "raw",
		Type:         "authenticated",
		Folder:       folder,
	})
	if err != nil {
		return "", "", err
	}
	return res.PublicID, res.Format, nil
}

func (c *Client) SignedDownloadURL(publicID, format string) (string, error) {
	expiresAt := time.Now().Add(5 * time.Minute)
	return c.cld.Upload.PrivateDownloadURL(uploader.PrivateDownloadURLParams{
		PublicID:     publicID,
		Format:       format,
		DeliveryType: "authenticated",
		ExpiresAt:    &expiresAt,
	})
}
