// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// Package s3cache provides an httpcache.Cache implementation that stores
// cached values on Amazon S3 with TTL support.
package s3cache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type cacheEntry struct {
	Data       []byte    `json:"data"`
	ExpiryTime time.Time `json:"expiry_time,omitempty"`
}

type cache struct {
	s3iface.S3API
	bucket, prefix string
	ttl            time.Duration
}

func (c *cache) Get(key string) ([]byte, bool) {
	key = path.Join(c.prefix, keyToFilename(key))
	input := &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	}

	resp, err := c.GetObject(input)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) && aerr.Code() != "NoSuchKey" {
			log.Printf("error fetching from s3: %v", aerr)
		}
		return nil, false
	}
	defer resp.Body.Close()

	var entry cacheEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		log.Printf("error decoding cache entry: %v", err)
		return nil, false
	}

	// Check if entry has expired
	if !entry.ExpiryTime.IsZero() && time.Now().After(entry.ExpiryTime) {
		go c.Delete(key) // Async cleanup
		return nil, false
	}

	return entry.Data, true
}

func (c *cache) Set(key string, value []byte) {
	key = path.Join(c.prefix, keyToFilename(key))

	entry := cacheEntry{
		Data: value,
	}

	// Only set expiry if TTL is configured
	if c.ttl > 0 {
		entry.ExpiryTime = time.Now().Add(c.ttl)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("error encoding cache entry: %v", err)
		return
	}

	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(bytes.NewReader(data)),
		Bucket: &c.bucket,
		Key:    &key,
	}

	_, err = c.PutObject(input)
	if err != nil {
		log.Printf("error writing to s3: %v", err)
	}
}

func (c *cache) Delete(key string) {
	key = path.Join(c.prefix, keyToFilename(key))
	input := &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	}

	_, err := c.DeleteObject(input)
	if err != nil {
		log.Printf("error deleting from s3: %v", err)
	}
}

func keyToFilename(key string) string {
	h := md5.New()
	_, _ = io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

// New constructs a cache configured using the provided URL string and TTL.
// URL should be of the form: "s3://region/bucket/optional-path-prefix".
func New(s string) (*cache, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	region := u.Host
	path := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	bucket := path[0]
	var prefix string
	if len(path) > 1 {
		prefix = path[1]
	}

	config := aws.NewConfig().WithRegion(region)

	// allow overriding some additional config options, mostly useful when
	// working with s3-compatible services other than AWS.
	if v := u.Query().Get("endpoint"); v != "" {
		config = config.WithEndpoint(v)
	}
	if v := u.Query().Get("disableSSL"); v == "1" {
		config = config.WithDisableSSL(true)
	}
	if v := u.Query().Get("s3ForcePathStyle"); v == "1" {
		config = config.WithS3ForcePathStyle(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	return &cache{
		S3API:  s3.New(sess),
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// NewWithTTL creates a new S3 cache with the specified TTL duration
func NewWithTTL(s string, ttl time.Duration) (*cache, error) {
	c, err := New(s)
	if err != nil {
		return nil, err
	}
	c.ttl = ttl
	return c, nil
}
