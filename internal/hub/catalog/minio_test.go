package catalog

import (
	"bytes"
	"context"
	"io"
	"testing"
)

// mockObjectStore implements ObjectStore for testing.
type mockObjectStore struct {
	uploaded map[string][]byte
	putErr   error
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{uploaded: make(map[string][]byte)}
}

func (m *mockObjectStore) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error {
	if m.putErr != nil {
		return m.putErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.uploaded[bucket+"/"+key] = data
	return nil
}

func (m *mockObjectStore) GetObjectURL(bucket, key string) string {
	return "http://minio:9000/" + bucket + "/" + key
}

func TestBinaryObjectKey(t *testing.T) {
	key := binaryObjectKey("ubuntu", "22.04", "curl_7.88.1_amd64.deb")
	want := "patches/ubuntu/22.04/curl_7.88.1_amd64.deb"
	if key != want {
		t.Errorf("binaryObjectKey() = %q, want %q", key, want)
	}
}

func TestUploadBinary(t *testing.T) {
	store := newMockObjectStore()
	data := []byte("fake-binary-content")

	key, checksum, err := uploadBinary(context.Background(), store, "patches", "ubuntu", "22.04", "curl.deb", bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("uploadBinary() error: %v", err)
	}
	if key != "patches/ubuntu/22.04/curl.deb" {
		t.Errorf("key = %q, want patches/ubuntu/22.04/curl.deb", key)
	}
	if checksum == "" {
		t.Error("checksum should not be empty")
	}
	if _, ok := store.uploaded["patches/patches/ubuntu/22.04/curl.deb"]; !ok {
		t.Error("binary was not uploaded to object store")
	}
}

func TestUploadBinaryError(t *testing.T) {
	store := newMockObjectStore()
	store.putErr = io.ErrUnexpectedEOF
	data := []byte("fake-binary-content")

	_, _, err := uploadBinary(context.Background(), store, "patches", "ubuntu", "22.04", "curl.deb", bytes.NewReader(data), int64(len(data)))
	if err == nil {
		t.Fatal("uploadBinary() should return error when PutObject fails")
	}
}
