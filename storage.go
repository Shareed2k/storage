package storage

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"path"
	"time"

	"github.com/gabriel-vasile/mimetype"
	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/fs/object"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/walk"
	"github.com/rs/xid"

	"github.com/shareed2k/storage/cache"
	"github.com/shareed2k/storage/fs"

	_ "github.com/rclone/rclone/backend/all" // import all backends
)

type (
	// Storage _
	Storage interface {
		Size(path string) int64
		Exists(path string) bool
		Hash(path string) string
		Delete(paths ...string) error
		URL(path string) (string, error)
		MakeDirectory(path string) error
		Get(name string) (fs.File, error)
		DeleteDirectory(path string) error
		LastModified(path string) time.Time
		AllDirectories(path string) []string
		AllFiles(path string) (files fs.Files)
		Files(path string, recursive ...bool) fs.Files
		Directories(path string, recursive ...bool) fs.Dirs
		TemporaryURL(path string, expire time.Duration) (string, error)
		Put(path string, in io.ReadCloser, metadata ...*fs.HTTPOption) (fs.File, error)
		PutFile(dir string, in io.ReadCloser, metadata ...*fs.HTTPOption) (fs.File, error)
	}

	storage struct {
		backend    rfs.Fs
		cacheStore cache.Store
		Config     *DiskConfig
	}
)

func init() {
	rfs.Config.LogLevel = rfs.LogLevelEmergency
	config.ConfigPath = "/dev/null"
}

// WithCacheDisk _
func WithCacheDisk(name string, store cache.Store) (Storage, error) {
	storage, err := newDisk(name)
	if err != nil {
		return nil, err
	}

	if store == nil {
		// use default cache,
		// for now i did'nt find a way to use it with another stores
		store = cache.New()
	}

	storage.cacheStore = store

	return storage, nil
}

// Disk _
func Disk(name string) (Storage, error) {
	return newDisk(name)
}

func newDisk(name string) (*storage, error) {
	cfg, err := getDiskConfig(name)
	if err != nil {
		return nil, err
	}

	// default timeout
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second * 30
	}

	regInfo, err := rfs.Find(cfg.Driver)
	if err != nil {
		return nil, err
	}

	//cm := fs.ConfigMap(regInfo, driver)
	cm := configmap.New()
	cm.AddGetter(cfg)

	backend, err := regInfo.NewFs(name, cfg.Root, cm)
	if err != nil {
		return nil, err
	}

	return &storage{
		backend: backend,
		Config:  cfg,
	}, nil
}

// not all backends support PublicLink
func (s *storage) URL(path string) (string, error) {
	// TODO: add expire default value to config
	return s.TemporaryURL(path, time.Minute*15)
}

// not all backends support PublicLink
func (s *storage) TemporaryURL(path string, expire time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	return operations.PublicLink(ctx, s.backend, path)
}

func (s *storage) PutFile(dir string, in io.ReadCloser, metadata ...*fs.HTTPOption) (fs.File, error) {
	body := &bytes.Buffer{}
	mime, err := mimetype.DetectReader(io.TeeReader(in, body))
	if err != nil {
		return nil, err
	}

	id := xid.New().String()
	extension := mime.Extension()

	o, err := s.put(path.Join(dir, id+extension), ioutil.NopCloser(body), metadata...)
	if err != nil {
		return nil, err
	}

	return fs.ObjectWrapper(o), nil
}

func (s *storage) Put(path string, in io.ReadCloser, metadata ...*fs.HTTPOption) (fs.File, error) {
	o, err := s.put(path, in, metadata...)
	if err != nil {
		return nil, err
	}

	return fs.ObjectWrapper(o), nil
}

func (s *storage) put(path string, in io.ReadCloser, metadata ...*fs.HTTPOption) (rfs.Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	var options []rfs.OpenOption
	for _, option := range metadata {
		options = append(options, option)
	}

	objInfo := object.NewStaticObjectInfo(path, time.Now(), -1, false, nil, nil)
	o, err := s.backend.Put(ctx, in, objInfo, options...)
	if err != nil {
		return nil, err
	}

	if s.cacheStore != nil {
		if err := s.cacheStore.Put(path, fs.ObjectWrapper(o)); err != nil {
			return nil, err
		}
	}

	return o, nil
}

func (s *storage) Delete(paths ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	delChan := make(rfs.ObjectsChan, rfs.Config.Transfers)
	delErr := make(chan error, 1)
	go func() {
		delErr <- operations.DeleteFiles(ctx, delChan)
	}()
	for _, p := range paths {
		if o, err := s.backend.NewObject(ctx, p); err == nil {
			delChan <- o
		}
	}
	close(delChan)

	return <-delErr
}

func (s *storage) Size(path string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	o, err := s.backend.NewObject(ctx, path)
	if err != nil {
		return 0
	}

	return o.Size()
}

func (s *storage) LastModified(path string) time.Time {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	o, err := s.backend.NewObject(ctx, path)
	if err != nil {
		return time.Time{}
	}

	return o.ModTime(ctx)
}

func (s *storage) Hash(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	o, err := s.backend.NewObject(ctx, path)
	if err != nil {
		return ""
	}

	sum, _ := o.Hash(ctx, hash.MD5)

	return sum
}

func (s *storage) Get(path string) (f fs.File, err error) {
	create := func(key string) (value interface{}, ok bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
		defer cancel()

		o, err := s.backend.NewObject(ctx, key)
		if err != nil {
			return nil, false, err
		}

		return fs.ObjectWrapper(o), true, nil
	}

	if s.cacheStore != nil {
		f, err = s.cacheStore.Get(path, create)
		if err != nil {
			return nil, err
		}

		return f, nil
	}

	value, ok, err := create(path)
	if err != nil && !ok {
		return nil, err
	}

	return value.(fs.File), nil
}

func (s *storage) Exists(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	if ok, _ := rfs.FileExists(ctx, s.backend, path); ok {
		return true
	}

	return false
}

func (s *storage) MakeDirectory(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	return operations.Mkdir(ctx, s.backend, path)
}

func (s *storage) DeleteDirectory(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	return operations.Rmdir(ctx, s.backend, path)
}

func (s *storage) AllDirectories(path string) []string {
	return s.Directories(path, true)
}

func (s *storage) Directories(path string, recursive ...bool) (dirs fs.Dirs) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	r := false
	if len(recursive) > 0 {
		r = recursive[0]
	}

	if err := walk.ListR(ctx, s.backend, path, false, operations.ConfigMaxDepth(r), walk.ListDirs, func(entries rfs.DirEntries) error {
		entries.ForDir(func(dir rfs.Directory) {
			if dir != nil {
				dirs = append(dirs, dir.Remote())
			}
		})
		return nil
	}); err != nil {
		return dirs
	}

	return dirs
}

func (s *storage) AllFiles(path string) (files fs.Files) {
	return s.Files(path, true)
}

func (s *storage) Files(path string, recursive ...bool) (files fs.Files) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	r := false
	if len(recursive) > 0 {
		r = recursive[0]
	}

	if err := walk.ListR(ctx, s.backend, path, false, operations.ConfigMaxDepth(r), walk.ListObjects, func(entries rfs.DirEntries) error {
		entries.ForObject(func(o rfs.Object) {
			files = append(files, fs.ObjectWrapper(o))
		})
		return nil
	}); err != nil {
		return files
	}

	return files
}

func (c DiskConfig) Get(key string) (value string, ok bool) {
	value, ok = c.BackendConfig[key]
	return value, ok
}
