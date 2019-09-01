package drivecache

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"google.golang.org/api/drive/v3"

	"willnorris.com/go/imageproxy"
)

var ctx = context.Background()

var errFileNotFound = errors.New("drive: file not found")

type cache struct {
	parentID string
	client   *drive.FilesService
}

func (c *cache) Get(key string) ([]byte, bool) {
	id, err := c.getFileID(key)
	if err != nil {
		if err != errFileNotFound {
			log.Printf("error finding file from drive: %v", err)
		}
		return nil, false
	}
	resp, err := c.client.Get(id).Download()
	if err != nil {
		log.Printf("error reading file from drive: %v", err)
		return nil, false
	}
	defer resp.Body.Close()

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading file from drive: %v", err)
		return nil, false
	}

	return value, true
}

func (c *cache) Set(key string, value []byte) {
	_, err := c.client.Create(&drive.File{
		Name:    keyToFilename(key),
		Parents: []string{c.parentID},
	}).Media(bytes.NewReader(value)).Do()
	if err != nil {
		log.Printf("error creating file in drive: %v", err)
	}
}

func (c *cache) Delete(key string) {
	id, err := c.getFileID(key)
	if err != nil {
		if err != errFileNotFound {
			log.Printf("error finding file from drive: %v", err)
		}
	}
	err = c.client.Delete(id).Do()
	if err != nil {
		log.Printf("error deleting file from drive: %v", err)
	}
}

func keyToFilename(key string) string {
	h := md5.New()
	io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *cache) getFileID(key string) (string, error) {
	query := fmt.Sprintf("'%s' in parents and name='%s'", c.parentID, keyToFilename(key))
	fileList, err := c.client.List().Q(query).Do()
	if err != nil {
		return "", err
	}
	if len(fileList.Files) == 0 {
		return "", errFileNotFound
	}
	return fileList.Files[0].Id, nil
}

// New constructs a Cache storing files in google drive
func New(folderID string) (imageproxy.Cache, error) {
	client, err := drive.NewService(ctx)
	if err != nil {
		return nil, err
	}

	// ensures that the folder exists, futher validation can be done by
	// checking the capabilities whether this is a folder and you havae
	// read, write, list and delte access to the childrens
	_, err = client.Files.Get(folderID).Do()
	if err != nil {
		return nil, err
	}
	return &cache{
		parentID: folderID,
		client:   drive.NewFilesService(client),
	}, nil
}
