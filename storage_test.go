package storage

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/rclone/rclone/backend/memory"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/stretchr/testify/assert"
)

func init() {
	AddDiskConfig("local", DiskConfig{
		Driver: "local",
		Root:   "./testdata",
	})
}

func TestStorage_Get(t *testing.T) {
	storage, err := Disk("local")
	assert.NoError(t, err)

	r, err := storage.Get("test.json")

	if assert.NoError(t, err) {
		reader, err := r.Open()
		assert.NoError(t, err)

		data, err := ioutil.ReadAll(reader)
		assert.NoError(t, err)

		defer reader.Close()

		var jsonData struct {
			Test string
		}

		err = json.Unmarshal(data, &jsonData)

		assert.NoError(t, err)
		assert.Equal(t, "data for test", jsonData.Test)
	}
}

func TestStorage_Exists(t *testing.T) {
	storage, err := Disk("local")
	assert.NoError(t, err)

	t.Run("file exist", func(t *testing.T) {
		ok := storage.Exists("test.json")
		assert.True(t, ok, "json file is missing it can't be")
	})

	t.Run("file is missing", func(t *testing.T) {
		ok := storage.Exists("test-missing.json")
		assert.False(t, ok, "json file is exists it can't be")
	})
}

func TestStorage_Hash(t *testing.T) {
	storage, err := Disk("local")
	assert.NoError(t, err)

	sum := storage.Hash("test.json")
	assert.Equal(t, "fa91bd9ee771c66e3171d0bc8da6de50", sum)
}

func TestStorage_PutAndDelete(t *testing.T) {
	f, err := memory.NewFs("test", "/", configmap.New())
	assert.NoError(t, err)

	content := "just a test ;)"
	storage := Storage(&storage{backend: f, Config: &DiskConfig{
		Timeout: time.Second,
	}})

	t.Run("Create a file", func(t *testing.T) {
		size, err := storage.Put("test2.json", ioutil.NopCloser(strings.NewReader(content)))
		assert.NoError(t, err)

		assert.Equal(t, int64(len(content)), size)
	})

	t.Run("Delete a file", func(t *testing.T) {
		err := storage.Delete("test2.json")
		assert.NoError(t, err)

		assert.False(t, storage.Exists("test2.json"))
	})
}

func TestStorage_TemporaryURL(t *testing.T) {
	storage, err := Disk("local")
	assert.NoError(t, err)

	url, err := storage.TemporaryURL("test2.json", time.Second*15)
	assert.Error(t, err)

	assert.Equal(t, "", url)
}

/*func TestStorage_MakeDirectory(t *testing.T) {
	f, err := memory.NewFs("test", "/", configmap.New())
	assert.NoError(t, err)

	storage := Storage(&storage{backend: f})

	err = storage.MakeDirectory("/jopka/popka")
	assert.NoError(t, err)

	e, err := f.List(context.Background(), "/")
	assert.NoError(t, err)

	var dirName string
	e.ForDir(func(dir fs.Directory) {
		if dir.Remote() == "jopka" {
			dirName = dir.Remote()
		}
	})

	assert.Equal(t, "popka", dirName)
}*/
