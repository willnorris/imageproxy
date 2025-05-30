// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package gcscache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"cloud.google.com/go/storage"
)

// mockObjectHandle implements objectHandle for testing
type mockObjectHandle struct {
	data      []byte
	exists    bool
	readErr   error
	writeErr  error
	deleteErr error
	writeData *bytes.Buffer
}

func (m *mockObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if !m.exists {
		return nil, storage.ErrObjectNotExist
	}
	return io.NopCloser(bytes.NewReader(m.data)), nil
}

func (m *mockObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	if m.writeData == nil {
		m.writeData = &bytes.Buffer{}
	}
	return &mockWriter{buf: m.writeData, err: m.writeErr}
}

func (m *mockObjectHandle) Delete(ctx context.Context) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.exists = false
	return nil
}

// mockWriter implements io.WriteCloser for testing
type mockWriter struct {
	buf *bytes.Buffer
	err error
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	return w.buf.Write(p)
}

func (w *mockWriter) Close() error {
	return w.err
}

// mockBucketHandle implements bucketHandle for testing
type mockBucketHandle struct {
	objects map[string]objectHandle
}

func (b *mockBucketHandle) Object(name string) objectHandle {
	if b.objects == nil {
		b.objects = make(map[string]objectHandle)
	}
	if obj, exists := b.objects[name]; exists {
		return obj
	}
	// Create a new mock object
	obj := &mockObjectHandle{exists: false}
	b.objects[name] = obj
	return obj
}

func TestCacheGetEmptyObject(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("empty-key"): &mockObjectHandle{
				data:   []byte{},
				exists: true,
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

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
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("test-key"): &mockObjectHandle{
				data:   testData,
				exists: true,
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

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
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("missing-key"): &mockObjectHandle{
				exists: false,
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

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
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("error-key"): &mockObjectHandle{
				readErr: errors.New("read error"),
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

	// Test that read errors return cache miss
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
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("read-error-key"): &mockObjectHandleWithReadError{},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

	data, ok := c.Get("read-error-key")
	if data != nil {
		t.Errorf("Get returned non-nil data for read error")
	}
	if ok {
		t.Errorf("Get returned ok = true for read error, expected false")
	}
}

// mockObjectHandleWithReadError implements objectHandle with a failing reader
type mockObjectHandleWithReadError struct{}

func (m *mockObjectHandleWithReadError) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return io.NopCloser(&errorReader{}), nil
}

func (m *mockObjectHandleWithReadError) NewWriter(ctx context.Context) io.WriteCloser {
	return &mockWriter{buf: &bytes.Buffer{}}
}

func (m *mockObjectHandleWithReadError) Delete(ctx context.Context) error {
	return nil
}

// mockObjectHandleWithCloseError implements objectHandle with a writer that fails on close
type mockObjectHandleWithCloseError struct{}

func (m *mockObjectHandleWithCloseError) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return nil, storage.ErrObjectNotExist
}

func (m *mockObjectHandleWithCloseError) NewWriter(ctx context.Context) io.WriteCloser {
	return &mockWriter{buf: &bytes.Buffer{}, err: errors.New("close error")}
}

func (m *mockObjectHandleWithCloseError) Delete(ctx context.Context) error {
	return nil
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestCacheSet(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: make(map[string]objectHandle),
	}

	c := NewWithBucket(bucket, "test-prefix")

	testData := []byte("test image data")
	c.Set("test-key", testData)

	// Verify data was written
	expectedKey := "test-prefix/" + keyToFilename("test-key")
	obj := bucket.objects[expectedKey]
	mockObj, ok := obj.(*mockObjectHandle)
	if !ok || mockObj == nil || mockObj.writeData == nil {
		t.Fatalf("Set did not create object")
	}
	if !bytes.Equal(mockObj.writeData.Bytes(), testData) {
		t.Errorf("Set did not write correct data, got %v, want %v", mockObj.writeData.Bytes(), testData)
	}
}

func TestCacheSetWriteError(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("test-key"): &mockObjectHandle{
				writeErr: errors.New("write error"),
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

	// Should not panic on error, just log it
	c.Set("test-key", []byte("data"))
}

func TestCacheSetCloseError(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: make(map[string]objectHandle),
	}

	// Create object that will fail on close
	bucket.objects["test-prefix/"+keyToFilename("test-key")] = &mockObjectHandleWithCloseError{}

	c := NewWithBucket(bucket, "test-prefix")

	// Should not panic on error, just log it
	c.Set("test-key", []byte("data"))
}

func TestCacheDelete(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("test-key"): &mockObjectHandle{
				exists: true,
				data:   []byte("test data"),
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

	c.Delete("test-key")

	// Verify object was marked as deleted
	expectedKey := "test-prefix/" + keyToFilename("test-key")
	obj := bucket.objects[expectedKey]
	mockObj, ok := obj.(*mockObjectHandle)
	if !ok || mockObj.exists {
		t.Errorf("Delete did not mark object as deleted")
	}
}

func TestCacheDeleteError(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			"test-prefix/" + keyToFilename("test-key"): &mockObjectHandle{
				deleteErr: errors.New("delete error"),
			},
		},
	}

	c := NewWithBucket(bucket, "test-prefix")

	// Should not panic on error, just log it
	c.Delete("test-key")
}

func TestCacheWithoutPrefix(t *testing.T) {
	bucket := &mockBucketHandle{
		objects: map[string]objectHandle{
			keyToFilename("no-prefix-key"): &mockObjectHandle{
				data:   []byte("data"),
				exists: true,
			},
		},
	}

	c := NewWithBucket(bucket, "")

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
