// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package s3cache

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// mockS3Client is a mock implementation of the S3 client interface
type mockS3Client struct {
	s3iface.S3API
	storage map[string][]byte
}

func newMockS3Client() *mockS3Client {
	return &mockS3Client{
		storage: make(map[string][]byte),
	}
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if data, ok := m.storage[*input.Key]; ok {
		return &s3.GetObjectOutput{
			Body: aws.ReadSeekCloser(bytes.NewReader(data)),
		}, nil
	}
	return nil, awserr.New("NoSuchKey", "The specified key does not exist.", nil)
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	m.storage[*input.Key] = data
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	delete(m.storage, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func TestS3Cache(t *testing.T) {
	mock := newMockS3Client()
	c := &cache{
		S3API:  mock,
		bucket: "test-bucket",
		prefix: "test-prefix",
		ttl:    1 * time.Second,
	}

	// Test basic set and get
	t.Run("Basic Set and Get", func(t *testing.T) {
		key := "test-key"
		data := []byte("test-data")

		c.Set(key, data)
		got, exists := c.Get(key)
		if !exists {
			t.Error("expected data to exist in cache")
		}
		if string(got) != string(data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	// Test expiration
	t.Run("Expiration", func(t *testing.T) {
		key := "expiring-key"
		data := []byte("expiring-data")

		c.Set(key, data)
		time.Sleep(2 * time.Second) // Wait for TTL to expire

		_, exists := c.Get(key)
		if exists {
			t.Error("expected data to be expired")
		}
	})

	// Test deletion
	t.Run("Delete", func(t *testing.T) {
		key := "delete-key"
		data := []byte("delete-data")

		c.Set(key, data)
		c.Delete(key)

		_, exists := c.Get(key)
		if exists {
			t.Error("expected data to be deleted")
		}
	})

	// Test no TTL
	t.Run("No TTL", func(t *testing.T) {
		noTTLMock := newMockS3Client()
		noTTLCache := &cache{
			S3API:  noTTLMock,
			bucket: "test-bucket",
			prefix: "test-prefix",
			ttl:    0,
		}

		key := "no-ttl-key"
		data := []byte("no-ttl-data")

		noTTLCache.Set(key, data)
		got, exists := noTTLCache.Get(key)
		if !exists {
			t.Error("expected data to exist in cache")
		}
		if string(got) != string(data) {
			t.Errorf("got %q, want %q", got, data)
		}

		// Wait longer than the previous TTL
		time.Sleep(2 * time.Second)
		got, exists = noTTLCache.Get(key)
		if !exists {
			t.Error("expected data to still exist in cache with no TTL")
		}
	})
}

func TestNewWithTTL(t *testing.T) {
	ttl := 24 * time.Hour
	c, err := NewWithTTL("s3://us-west-2/test-bucket/test-prefix", ttl)
	if err != nil {
		t.Fatalf("NewWithTTL failed: %v", err)
	}

	if c.ttl != ttl {
		t.Errorf("got TTL %v, want %v", c.ttl, ttl)
	}
	if c.bucket != "test-bucket" {
		t.Errorf("got bucket %q, want %q", c.bucket, "test-bucket")
	}
	if c.prefix != "test-prefix" {
		t.Errorf("got prefix %q, want %q", c.prefix, "test-prefix")
	}
}
