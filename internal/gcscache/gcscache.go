// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// Package gcscache provides an httpcache.Cache implementation that stores
// cached values on Google Cloud Storage.
package gcscache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"path"

	"cloud.google.com/go/storage"
)

var ctx = context.Background()

// objectHandle interface wraps the methods we use from storage.ObjectHandle
type objectHandle interface {
	NewReader(ctx context.Context) (io.ReadCloser, error)
	NewWriter(ctx context.Context) io.WriteCloser
	Delete(ctx context.Context) error
}

// bucketHandle interface wraps the methods we use from storage.BucketHandle
type bucketHandle interface {
	Object(name string) objectHandle
}

// gcsObjectHandle wraps storage.ObjectHandle to implement our interface
type gcsObjectHandle struct {
	*storage.ObjectHandle
}

func (o *gcsObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return o.ObjectHandle.NewReader(ctx)
}

func (o *gcsObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	return o.ObjectHandle.NewWriter(ctx)
}

func (o *gcsObjectHandle) Delete(ctx context.Context) error {
	return o.ObjectHandle.Delete(ctx)
}

// gcsBucketHandle wraps storage.BucketHandle to implement our interface
type gcsBucketHandle struct {
	*storage.BucketHandle
}

func (b *gcsBucketHandle) Object(name string) objectHandle {
	return &gcsObjectHandle{b.BucketHandle.Object(name)}
}

type cache struct {
	bucket bucketHandle
	prefix string
}

func (c *cache) Get(key string) ([]byte, bool) {
	r, err := c.object(key).NewReader(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			log.Printf("error reading from gcs: %v", err)
		}
		return nil, false
	}
	defer r.Close()

	value, err := io.ReadAll(r)
	if err != nil {
		log.Printf("error reading from gcs: %v", err)
		return nil, false
	}

	// Treat empty objects as cache misses to prevent cache poisoning
	// when objects are deleted by lifecycle rules
	if len(value) == 0 {
		return nil, false
	}

	return value, true
}

func (c *cache) Set(key string, value []byte) {
	w := c.object(key).NewWriter(ctx)
	if _, err := w.Write(value); err != nil {
		log.Printf("error writing to gcs: %v", err)
	}
	if err := w.Close(); err != nil {
		log.Printf("error closing gcs object writer: %v", err)
	}
}

func (c *cache) Delete(key string) {
	if err := c.object(key).Delete(ctx); err != nil {
		log.Printf("error deleting gcs object: %v", err)
	}
}

func (c *cache) object(key string) objectHandle {
	name := path.Join(c.prefix, keyToFilename(key))
	return c.bucket.Object(name)
}

func keyToFilename(key string) string {
	h := md5.New()
	_, _ = io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

// NewWithBucket constructs a cache using the provided bucket handle
func NewWithBucket(bucket bucketHandle, prefix string) *cache {
	return &cache{
		bucket: bucket,
		prefix: prefix,
	}
}

// New constructs a Cache storing files in the specified GCS bucket.  If prefix
// is not empty, objects will be prefixed with that path. Credentials should
// be specified using one of the mechanisms supported for Application Default
// Credentials (see https://cloud.google.com/docs/authentication/production)
func New(bucket, prefix string) (*cache, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &cache{
		prefix: prefix,
		bucket: &gcsBucketHandle{client.Bucket(bucket)},
	}, nil
}
