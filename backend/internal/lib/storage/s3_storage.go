package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps a minio.Client scoped to one configured bucket. Works against
// any S3-compatible provider — MinIO, Cloudflare R2, Backblaze B2, AWS S3 —
// by pointing Endpoint at the provider's host.
type Client struct {
	mc     *minio.Client
	bucket string
}

func NewClient(cfg *config.ObjectStorageConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create object storage client: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

// EnsureBucket creates the configured bucket if it doesn't already exist.
// Safe to call on every startup.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{Region: ""}); err != nil {
			return fmt.Errorf("failed to create bucket %q: %w", c.bucket, err)
		}
	}
	return nil
}

// Upload streams reader to key. size may be -1 if unknown.
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	span := sentry.StartSpan(ctx, "storage.upload", sentry.WithDescription(key))
	span.SetData("storage.bucket", c.bucket)
	defer span.Finish()

	if _, err := c.mc.PutObject(span.Context(), c.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", err.Error())
		return fmt.Errorf("failed to upload object %q: %w", key, err)
	}
	span.Status = sentry.SpanStatusOK
	return nil
}

// Download returns a reader for key. The caller must Close it.
func (c *Client) Download(ctx context.Context, key string) (*minio.Object, error) {
	span := sentry.StartSpan(ctx, "storage.download", sentry.WithDescription(key))
	span.SetData("storage.bucket", c.bucket)
	defer span.Finish()

	obj, err := c.mc.GetObject(span.Context(), c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", err.Error())
		return nil, fmt.Errorf("failed to download object %q: %w", key, err)
	}
	span.Status = sentry.SpanStatusOK
	return obj, nil
}

// Delete removes key from the bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	span := sentry.StartSpan(ctx, "storage.delete", sentry.WithDescription(key))
	span.SetData("storage.bucket", c.bucket)
	defer span.Finish()

	if err := c.mc.RemoveObject(span.Context(), c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", err.Error())
		return fmt.Errorf("failed to delete object %q: %w", key, err)
	}
	span.Status = sentry.SpanStatusOK
	return nil
}

// PresignedGetURL returns a temporary, publicly-fetchable download URL.
func (c *Client) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to presign url for %q: %w", key, err)
	}
	return u.String(), nil
}

// Ping verifies connectivity for use in health checks.
func (c *Client) Ping(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("object storage unreachable: %w", err)
	}
	if !exists {
		return fmt.Errorf("configured bucket %q does not exist", c.bucket)
	}
	return nil
}
