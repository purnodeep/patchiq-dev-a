package catalog

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStore implements ObjectStore backed by a MinIO (S3-compatible) server.
type MinIOStore struct {
	client   *minio.Client
	endpoint string
	useSSL   bool
}

// NewMinIOStore creates a MinIOStore connected to the given endpoint.
// It ensures the target bucket exists on first use.
func NewMinIOStore(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio store: connect to %s: %w", endpoint, err)
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return &MinIOStore{
		client:   client,
		endpoint: scheme + "://" + endpoint,
		useSSL:   useSSL,
	}, nil
}

// EnsureBucket creates the bucket if it does not already exist.
func (s *MinIOStore) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("minio store: check bucket %s: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("minio store: create bucket %s: %w", bucket, err)
	}
	return nil
}

// PutObject uploads the contents of reader to the given bucket and key.
func (s *MinIOStore) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("minio store: put object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// GetObjectURL returns the direct URL to an object.
// For internal-only use; agents access binaries via the server's file server, not directly.
func (s *MinIOStore) GetObjectURL(bucket, key string) string {
	return fmt.Sprintf("%s/%s/%s", s.endpoint, bucket, key)
}

// GetObject returns a ReadCloser for the object at bucket/key along with its size.
// Returns -1 for size if unknown. The caller must close the returned io.ReadCloser.
func (s *MinIOStore) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, -1, fmt.Errorf("minio store: get object %s/%s: %w", bucket, key, err)
	}
	info, err := obj.Stat()
	if err != nil {
		// Non-fatal: we can still stream without a known size.
		return obj, -1, nil
	}
	return obj, info.Size, nil
}
