package main

import (
	"io/ioutil"
	"log"

	"github.com/shareed2k/storage"
	"github.com/shareed2k/storage/cache"
	"github.com/shareed2k/storage/fs"
)

func main() {
	storage.AddDiskConfig("gcs", storage.DiskConfig{
		Driver: "gcs",
		Root:   "for_test_only_roman",
	})

	s, err := storage.Disk("local")
	if err != nil {
		log.Fatal(err)
	}

	gcs, err := storage.WithCacheDisk("gcs", cache.New())
	if err != nil {
		log.Fatal(err)
	}

	f, err := s.Get("/config.go")
	if err != nil {
		log.Fatal(err)
	}

	reader, err := f.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer reader.Close()

	if _, err := gcs.Put("/config.go", reader, &fs.HTTPOption{
		Key:   "content-encoding",
		Value: "foo",
	}); err != nil {
		log.Fatal(err)
	}

	f, err = gcs.Get("/config.go")
	if err != nil {
		log.Fatal(err)
	}

	rr, err := f.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer rr.Close()

	data, err := ioutil.ReadAll(rr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("data ====> ", string(data))

	/*if err := s.Delete("/tmp/bringger.50787.sock3", "/tmp/bringger.50787.sock2"); err != nil {
		log.Fatal(err)
	}*/

	if err := gcs.Files("").ForError(func(f fs.File) error {
		log.Println("file ===> ", f.Stat().Hash())
		log.Println("file ===> ", f.Stat().Name())

		return nil
	}); err != nil {
		log.Fatal(err)
	}

	/*gcs, err := storage.Disk("gcs")
	if err != nil {
		log.Fatal(err)
	}

	size, err := s.Put("/tmp/bringger.50787.sock3", ioutil.NopCloser(strings.NewReader("hello world")))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("size ==> ", size)

	if ok := gcs.Exists("/bringger.50787.sock"); ok {
		log.Println("file size ===> ", gcs.Size("/bringger.50787.sock"))
		log.Println("file hash ===> ", gcs.Hash("/bringger.50787.sock"))
		log.Println("file time ===> ", gcs.LastModified("/bringger.50787.sock"))
	}

	if err := gcs.MakeDirectory("test2"); err != nil {
		log.Fatal("create gsc dir ===> ", err)
	}

	log.Println("gcs dirs ===> ", gcs.Directories(""))
	log.Println("dirs ===> ", s.AllDirectories("/tmp"))

	if ok := s.Exists("/Users/shareed2k/schema.json"); ok {
		log.Println("file size ===> ", s.Size("/Users/shareed2k/schema.json"))
		log.Println("file hash ===> ", s.Hash("/Users/shareed2k/schema.json"))
		log.Println("file time ===> ", s.LastModified("/Users/shareed2k/schema.json"))
	}*/
}
