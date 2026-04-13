package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
)

// ObjectStore abstracts object storage operations (MinIO, S3, etc.).
type ObjectStore interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error
	GetObjectURL(bucket, key string) string
}

// binaryObjectKey builds the MinIO key: patches/{os_family}/{os_version}/{filename}.
func binaryObjectKey(osFamily, osVersion, filename string) string {
	return path.Join("patches", osFamily, osVersion, filename)
}

// uploadBinaryWithKey uploads a binary to the object store under an explicit key and returns
// the key and SHA256 checksum. Use this when the caller controls the full key path.
func uploadBinaryWithKey(ctx context.Context, store ObjectStore, bucket, key string, reader io.Reader, size int64) (string, string, error) {
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := store.PutObject(ctx, bucket, key, tee, size); err != nil {
		return "", "", fmt.Errorf("upload binary %s: %w", key, err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return key, checksum, nil
}

// uploadBinary uploads a binary to the object store and returns the key and SHA256 checksum.
// The reader is consumed once: checksum is computed during upload via a TeeReader.
func uploadBinary(ctx context.Context, store ObjectStore, bucket, osFamily, osVersion, filename string, reader io.Reader, size int64) (string, string, error) {
	key := binaryObjectKey(osFamily, osVersion, filename)

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := store.PutObject(ctx, bucket, key, tee, size); err != nil {
		return "", "", fmt.Errorf("upload binary %s: %w", key, err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return key, checksum, nil
}
