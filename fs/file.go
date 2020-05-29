package fs

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/hash"
)

var ErrStop = errors.New("stop iter")

type (
	File interface {
		Name() string
		Stat() FileInfo
		Open() (io.ReadCloser, error)
	}

	FileInfo interface {
		Name() string
		Hash() string
		Size() int64
		ModTime() time.Time
	}

	file struct {
		object fs.Object
	}

	fileInfo struct {
		object fs.Object
	}

	Files []File
)

func ObjectWrapper(o fs.Object) File {
	return &file{object: o}
}

func (f file) Name() string {
	return f.object.Remote()
}

func (f file) Stat() FileInfo {
	return &fileInfo{object: f.object}
}

func (f file) Open() (io.ReadCloser, error) {
	return f.object.Open(context.Background())
}

func (f *file) Update(in io.ReadCloser) error {
	return f.object.Update(context.Background(), in, f.object)
}

func (f fileInfo) Name() string {
	return f.object.Remote()
}

func (f fileInfo) Hash() string {
	sum, _ := f.object.Hash(context.Background(), hash.MD5)
	return sum
}

func (f fileInfo) Size() int64 {
	return f.object.Size()
}

func (f fileInfo) ModTime() time.Time {
	return f.object.ModTime(context.Background())
}

func (f Files) ForFileError(fn func(f File) error) error {
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
