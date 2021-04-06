// Package osscache provides an httpcache.Cache implementation that stores
// cached values on Aliyun OSS.
package osscache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type cache struct {
	bucket *oss.Bucket
	prefix string
}

func (c *cache) Get(key string) ([]byte, bool) {
	r, err := c.bucket.GetObject(c.objectKey(key))
	if err != nil {
		return nil, false
	}
	defer r.Close()

	value, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("error reading from aliyun oss: %v", err)
		return nil, false
	}

	return value, true
}

func (c *cache) Set(key string, value []byte) {
	if err := c.bucket.PutObject(c.objectKey(key), bytes.NewReader(value)); err != nil {
		log.Printf("error writing to aliyun oss: %v", err)
	}
}

func (c *cache) Delete(key string) {
	if err := c.bucket.DeleteObject(c.objectKey(key)); err != nil {
		log.Printf("error deleting aliyun oss object: %v", err)
	}
}

func (c *cache) objectKey(key string) string {
	return path.Join(c.prefix, keyToFilename(key))
}

func keyToFilename(key string) string {
	h := md5.New()
	_, _ = io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

// New constructs a Cache storing files in the specified GCS bucket.  If prefix
// is not empty, objects will be prefixed with that path. Credentials should
// be specified using one of the mechanisms supported for Application Default
// Credentials (see https://cloud.google.com/docs/authentication/production)
func New(s string) (*cache, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimPrefix(u.Path, "/")

	client, err := oss.New(
		u.Query().Get("endpoint"),
		os.Getenv("ALIYUN_ACCESS_KEY_ID"),
		os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
	)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(u.Host)
	if err != nil {
		return nil, err
	}

	return &cache{
		prefix: prefix,
		bucket: bucket,
	}, nil
}
