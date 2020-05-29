package cache

import (
	"errors"

	"github.com/rclone/rclone/lib/cache"

	"github.com/shareed2k/storage/fs"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

type memory struct {
	*cache.Cache
}

func New() *memory {
	return &memory{Cache: cache.New()}
}

func (c *memory) Clear() {
	c.Cache.Clear()
}

func (c *memory) Entries() int {
	return c.Cache.Entries()
}

func (c *memory) Put(path string, file fs.File) {
	c.Cache.Put(path, file)
}

func (c *memory) Rename(oldPath, newPath string) (file fs.File, found bool) {
	v, found := c.Cache.Rename(oldPath, newPath)
	if !found {
		return nil, false
	}

	if file, found = v.(fs.File); found {
		return file, true
	}

	return nil, false
}

func (c *memory) Get(path string, create CreateFunc) (file fs.File, err error) {
	v, err := c.Cache.Get(path, func(key string) (value interface{}, ok bool, error error) {
		return create(key)
	})
	if err != nil {
		return nil, err
	}

	if file, found := v.(fs.File); found {
		return file, nil
	}

	return nil, ErrFileNotFound
}
