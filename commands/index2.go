package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/nicksellen/audiotags"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type WalkFunc3 func(path string)

func Index2(root string) {

	dbpath := "db.bleve"

	mapping := bleve.NewIndexMapping()

	var index bleve.Index
	index, err := bleve.New(dbpath, mapping)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("importing from %s\n", root)

	var filecount uint32 = 0
	var processedcount uint32 = 0

	batch := bleve.NewBatch()

	var wg sync.WaitGroup

	q := make(chan bleve.Batch)

	go func() {
		for batch := range q {
			fmt.Printf("indexing batch\n")
			err := index.Batch(batch)
			if err != nil {
				log.Fatal(err)
			}
			wg.Done()
		}
	}()

	walk(root, func(path string) {
		atomic.AddUint32(&filecount, 1)

		props, audioProps, err := audiotags.Read(root + path)
		atomic.AddUint32(&processedcount, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %s\n", path, err)
		} else {
			props["length"] = strconv.Itoa(audioProps.Length)
			props["bitrate"] = strconv.Itoa(audioProps.Bitrate)
			props["samplerate"] = strconv.Itoa(audioProps.Samplerate)
			props["channels"] = strconv.Itoa(audioProps.Channels)

			batch.Index(path, props)

			num := atomic.LoadUint32(&processedcount)

			if num%100 == 0 {
				fmt.Printf("read %d entries\n", atomic.LoadUint32(&processedcount))
				wg.Add(1)
				q <- batch
				batch = bleve.NewBatch()

				fmt.Fprintf(os.Stderr, "%d/%d\n", num, atomic.LoadUint32(&filecount))
			}

		}
	})
	fmt.Printf("waiting for indexing to finish\n")
	wg.Wait()
	fmt.Fprintf(os.Stderr, "indexed %d files\n", atomic.LoadUint32(&processedcount))

}

func walk(root string, fn WalkFunc3) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			path := strings.TrimPrefix(path, root)
			lower := strings.ToLower(path)
			if strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".m4a") {
				fn(path)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func CalculateSha1Sum(filename string) string {
	h := sha1.New()
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}
