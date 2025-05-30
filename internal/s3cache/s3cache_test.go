// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package s3cache

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// mockS3Client implements s3iface.S3API for testing
type mockS3Client struct {
	s3iface.S3API
	objects map[string][]byte
	getErr  error
	putErr  error
	delErr  error
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	key := aws.StringValue(input.Key)
	data, exists := m.objects[key]
	if !exists {
		return nil, awserr.New("NoSuchKey", "The specified key does not exist.", nil)
	}

	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(data)),
	}, nil
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if m.putErr != nil {
		return nil, m.putErr
	}

	if m.objects == nil {
		m.objects = make(map[string][]byte)
	}

	key := aws.StringValue(input.Key)
	data, _ := io.ReadAll(input.Body)
	m.objects[key] = data

	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	if m.delErr != nil {
		return nil, m.delErr
	}

	key := aws.StringValue(input.Key)
	delete(m.objects, key)

	return &s3.DeleteObjectOutput{}, nil
}

func TestCacheGetEmptyObject(t *testing.T) {
	mock := &mockS3Client{
		objects: map[string][]byte{
			"test-prefix/" + keyToFilename("empty-key"): []byte{},
		},
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Test that empty objects are treated as cache misses
	data, ok := c.Get("empty-key")
	if data != nil {
		t.Errorf("Get returned non-nil data for empty object")
	}
	if ok {
		t.Errorf("Get returned ok = true for empty object, expected false")
	}
}

func TestCacheGetNonEmptyObject(t *testing.T) {
	testData := []byte("test image data")
	mock := &mockS3Client{
		objects: map[string][]byte{
			"test-prefix/" + keyToFilename("test-key"): testData,
		},
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Test that non-empty objects are returned correctly
	data, ok := c.Get("test-key")
	if !bytes.Equal(data, testData) {
		t.Errorf("Get returned incorrect data, got %v, want %v", data, testData)
	}
	if !ok {
		t.Errorf("Get returned ok = false for existing object, expected true")
	}
}

func TestCacheGetMissingObject(t *testing.T) {
	mock := &mockS3Client{
		objects: make(map[string][]byte),
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Test that missing objects return cache miss
	data, ok := c.Get("missing-key")
	if data != nil {
		t.Errorf("Get returned non-nil data for missing object")
	}
	if ok {
		t.Errorf("Get returned ok = true for missing object, expected false")
	}
}

func TestCacheGetError(t *testing.T) {
	mock := &mockS3Client{
		getErr: awserr.New("InternalError", "Internal server error", nil),
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Test that S3 errors (other than NoSuchKey) return cache miss
	data, ok := c.Get("error-key")
	if data != nil {
		t.Errorf("Get returned non-nil data for error case")
	}
	if ok {
		t.Errorf("Get returned ok = true for error case, expected false")
	}
}

func TestCacheGetReadError(t *testing.T) {
	// Create a custom mock that returns a reader that fails
	mock := &mockS3ClientWithReadError{}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	data, ok := c.Get("read-error-key")
	if data != nil {
		t.Errorf("Get returned non-nil data for read error")
	}
	if ok {
		t.Errorf("Get returned ok = true for read error, expected false")
	}
}

// mockS3ClientWithReadError implements s3iface.S3API with a failing reader
type mockS3ClientWithReadError struct {
	s3iface.S3API
}

func (m *mockS3ClientWithReadError) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{
		Body: io.NopCloser(&errorReader{}),
	}, nil
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestCacheSet(t *testing.T) {
	mock := &mockS3Client{
		objects: make(map[string][]byte),
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	testData := []byte("test image data")
	c.Set("test-key", testData)

	// Verify data was stored
	expectedKey := "test-prefix/" + keyToFilename("test-key")
	storedData := mock.objects[expectedKey]
	if !bytes.Equal(storedData, testData) {
		t.Errorf("Set did not store correct data, got %v, want %v", storedData, testData)
	}
}

func TestCacheSetError(t *testing.T) {
	mock := &mockS3Client{
		putErr: errors.New("put error"),
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Should not panic on error, just log it
	c.Set("test-key", []byte("data"))
}

func TestCacheDelete(t *testing.T) {
	mock := &mockS3Client{
		objects: map[string][]byte{
			"test-prefix/" + keyToFilename("test-key"): []byte("test data"),
		},
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	c.Delete("test-key")

	// Verify data was deleted
	expectedKey := "test-prefix/" + keyToFilename("test-key")
	if _, exists := mock.objects[expectedKey]; exists {
		t.Errorf("Delete did not remove object from cache")
	}
}

func TestCacheDeleteError(t *testing.T) {
	mock := &mockS3Client{
		delErr: errors.New("delete error"),
	}

	c := NewWithClient(mock, "test-bucket", "test-prefix")

	// Should not panic on error, just log it
	c.Delete("test-key")
}

func TestCacheWithoutPrefix(t *testing.T) {
	mock := &mockS3Client{
		objects: map[string][]byte{
			keyToFilename("no-prefix-key"): []byte("data"),
		},
	}

	c := NewWithClient(mock, "test-bucket", "")

	data, ok := c.Get("no-prefix-key")
	if !bytes.Equal(data, []byte("data")) {
		t.Errorf("Get with no prefix returned incorrect data")
	}
	if !ok {
		t.Errorf("Get with no prefix returned ok = false, expected true")
	}
}

func TestKeyToFilename(t *testing.T) {
	// Test that keyToFilename produces consistent results
	key1 := "test-key-1"
	key2 := "test-key-2"

	filename1 := keyToFilename(key1)
	filename2 := keyToFilename(key2)

	// Same key should produce same filename
	if keyToFilename(key1) != filename1 {
		t.Errorf("keyToFilename not consistent for same key")
	}

	// Different keys should produce different filenames
	if filename1 == filename2 {
		t.Errorf("keyToFilename produced same filename for different keys")
	}

	// Filename should be hex encoded MD5
	if len(filename1) != 32 { // MD5 produces 16 bytes = 32 hex chars
		t.Errorf("keyToFilename produced unexpected length: %d", len(filename1))
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		bucket  string
		prefix  string
	}{
		{
			name:   "basic URL",
			url:    "s3://us-east-1/my-bucket",
			bucket: "my-bucket",
			prefix: "",
		},
		{
			name:   "URL with prefix",
			url:    "s3://us-west-2/my-bucket/cache/prefix",
			bucket: "my-bucket",
			prefix: "cache/prefix",
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if c.bucket != tt.bucket {
					t.Errorf("New() bucket = %v, want %v", c.bucket, tt.bucket)
				}
				if c.prefix != tt.prefix {
					t.Errorf("New() prefix = %v, want %v", c.prefix, tt.prefix)
				}
			}
		})
	}
}
