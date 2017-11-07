// Package s3cache provides an httpcache.Cache implementation that stores
// cached values on Amazon S3.
package s3cache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type cache struct {
	*s3.S3
	bucket, prefix string
}

func (c *cache) Get(key string) ([]byte, bool) {
	key = path.Join(c.prefix, keyToFilename(key))
	input := &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	}

	resp, err := c.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != "NoSuchKey" {
			log.Printf("error fetching from s3: %v", aerr)
		}
		return nil, false
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading s3 response body: %v", err)
		return nil, false
	}

	return value, true
}
func (c *cache) Set(key string, value []byte) {
	key = path.Join(c.prefix, keyToFilename(key))
	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(bytes.NewReader(value)),
		Bucket: &c.bucket,
		Key:    &key,
	}

	_, err := c.PutObject(input)
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
	io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

// New constructs a cache configured using the provided URL string.  URL should
// be of the form: "s3://region/bucket/optional-path-prefix".  Credentials
// should be specified using one of the mechanisms supported by aws-sdk-go (see
// https://docs.aws.amazon.com/sdk-for-go/api/aws/session/).
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

	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		return nil, err
	}

	return &cache{
		S3:     s3.New(sess),
		bucket: bucket,
		prefix: prefix,
	}, nil
}
