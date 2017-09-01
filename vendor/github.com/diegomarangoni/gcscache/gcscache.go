package gcscache

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type Cache struct {
	pathPrefix string
	bucket     *storage.BucketHandle
}

func (c *Cache) Get(key string) (resp []byte, ok bool) {
	obj := c.bucket.Object(c.url(key))

	rdr, err := obj.NewReader(context.Background())
	if err != nil {
		return []byte{}, false
	}
	defer rdr.Close()

	resp, err = ioutil.ReadAll(rdr)
	if err != nil {
		log.Printf("gcscache.Get failed: %s", err)
	}

	return resp, err == nil
}

func (c *Cache) Set(key string, resp []byte) {
	obj := c.bucket.Object(c.url(key))

	contentType := http.DetectContentType(resp)

	w := obj.NewWriter(context.Background())
	w.ContentType = contentType
	w.ObjectAttrs.ContentType = contentType

	_, err := w.Write(resp)
	if err != nil {
		log.Printf("gcscache.Set failed: %s", err)
	}

	err = w.Close()
	if err != nil {
		log.Printf("gcscache.Set failed: %s", err)
	}
}

func (c *Cache) Delete(key string) {
	obj := c.bucket.Object(c.url(key))

	err := obj.Delete(context.Background())
	if err != nil {
		log.Printf("gcscache.Delete failed: %s", err)
	}
}

func (c *Cache) url(key string) string {
	key = cacheKeyToObjectKey(key)
	if strings.HasSuffix(c.pathPrefix, "/") {
		return c.pathPrefix + key
	}
	return c.pathPrefix + "/" + key
}

func cacheKeyToObjectKey(key string) string {
	h := md5.New()
	io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

func New(bucketURL string) *Cache {
	cfg, err := google.JWTConfigFromJSON([]byte(os.Getenv("GCP_PRIVATE_KEY")), storage.ScopeReadWrite)
	if err != nil {
		panic(err)
	}
	ts := cfg.TokenSource(context.Background())
	opt := option.WithTokenSource(ts)

	client, err := storage.NewClient(context.Background(), opt)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("gcs://([^/]+)(/(.+)?)?$")
	if !r.MatchString(bucketURL) {
		panic("Invalid bucket string format. Must match: gcs://bucket-name/path/prefix")
	}

	match := r.FindStringSubmatch(bucketURL)

	bucketName := match[1]
	pathPrefix := match[3]

	return &Cache{
		pathPrefix: pathPrefix,
		bucket:     client.Bucket(bucketName),
	}
}
