package fs

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/fs/operations"
)

var ErrStop = errors.New("stop iter")

type (
	File interface {
		Path() string
		Stat() FileInfo
		Move(newPath string) error
		Copy(newPath string) error
		Open() (io.ReadCloser, error)
		Update(in io.ReadCloser, metadata ...*HTTPOption) error
	}

	FileInfo interface {
		Name() string
		Hash() string
		Size() int64
		ModTime() time.Time
	}

	HTTPOption = fs.HTTPOption

	file struct {
		object fs.Object
	}

	fileInfo struct {
		object fs.Object
	}

	Files []File

	Dirs []string
)

func ObjectWrapper(o fs.Object) File {
	return &file{object: o}
}

func (f *file) Path() string {
	return f.object.Remote()
}

func (f *file) Stat() FileInfo {
	return &fileInfo{object: f.object}
}

func (f *file) Open() (io.ReadCloser, error) {
	return f.object.Open(context.Background())
}

func (f *file) Update(in io.ReadCloser, metadata ...*HTTPOption) error {
	var options []fs.OpenOption
	for _, option := range metadata {
		options = append(options, option)
	}

	return f.object.Update(context.Background(), in, f.object, options...)
}

func (f *file) Copy(newPath string) error {
	backend := f.object.Fs().(fs.Fs)
	newO, err := operations.Copy(context.Background(), backend, nil, newPath, f.object)
	if err != nil {
		return err
	}

	f.object = newO

	return nil
}

func (f *file) Move(newPath string) error {
	backend := f.object.Fs().(fs.Fs)
	newO, err := operations.Move(context.Background(), backend, nil, newPath, f.object)
	if err != nil {
		return err
	}

	f.object = newO

	return nil
}

func (f *fileInfo) Name() string {
	return filepath.Base(f.object.Remote())
}

func (f *fileInfo) Hash() string {
	sum, _ := f.object.Hash(context.Background(), hash.MD5)
	return sum
}

func (f *fileInfo) Size() int64 {
	return f.object.Size()
}

func (f *fileInfo) ModTime() time.Time {
	return f.object.ModTime(context.Background())
}

func (f Files) ForError(fn func(f File) error) error {
	for _, file := range f {
		if err := fn(file); err != nil {
			if errors.Is(err, ErrStop) {
				return nil
			}

			return err
		}
	}

	return nil
}

func (d Dirs) ForError(fn func(d string) error) error {
	for _, dir := range d {
		if err := fn(dir); err != nil {
			if errors.Is(err, ErrStop) {
				return nil
			}

			return err
		}
	}

	return nil
}
