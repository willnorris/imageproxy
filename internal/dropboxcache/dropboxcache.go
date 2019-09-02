package dropboxcache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"willnorris.com/go/imageproxy"
)

type cache struct {
	client files.Client
	folder string
}

func (c *cache) Get(key string) ([]byte, bool) {
	_, r, err := c.client.Download(files.NewDownloadArg(c.keyToFilePath(key)))
	if err != nil {
		log.Printf("error reteriving from dropbox: %v", err)
		return nil, false
	}
	value, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("error reading from dropbox: %v", err)
		return nil, false
	}
	return value, true
}

func (c *cache) Set(key string, value []byte) {

	commitInfo := files.NewCommitInfo(c.keyToFilePath(key))
	commitInfo.Mode.Tag = "overwrite"
	_, err := c.client.Upload(commitInfo, bytes.NewReader(value))
	if err != nil {
		log.Printf("error writing to dropbox: %v", err)
	}
}

func (c *cache) Delete(key string) {
	_, err := c.client.Delete(files.NewDeleteArg(c.keyToFilePath(key)))
	if err != nil {
		log.Printf("error deleting from dropbox: %v", err)
	}
}

func (c *cache) keyToFilePath(key string) string {
	h := md5.New()
	io.WriteString(h, key)
	return path.Join(c.folder, strings.ToLower(hex.EncodeToString(h.Sum(nil))))
}

// New constructs a new cache that stores data in a dropbox folder
func New(folder string) (imageproxy.Cache, error) {
	config := dropbox.Config{
		Token: os.Getenv("DROPBOX_ACCESS_TOKEN"),
	}

	client := files.New(config)
	folder = strings.ToLower(path.Join("/", folder))

	if len(folder) > 1 {
		metadata, err := client.GetMetadata(files.NewGetMetadataArg(folder))

		if err != nil {
			return nil, err
		}

		if _, ok := metadata.(*files.FolderMetadata); !ok {
			return nil, errors.New("dropbox: not a folder: " + folder)
		}
	}

	return &cache{
		client: client,
		folder: folder,
	}, nil
}
