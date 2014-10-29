package commands

import (
	"code.google.com/p/go-uuid/uuid"
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nicksellen/audioplayer/processing"
	"github.com/nicksellen/audiotags"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type WalkFunc func(path string, info os.FileInfo)

func Import(importdir string) {

	db, err := sql.Open("sqlite3", "./db.v2.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSql := `

    create table if not exists stores (
      id text primary key
    );

    create table if not exists files (
      id          text primary key,
      track_hash  text,
      change_hash text,
      store_id    integer,
      path        text,
      dir_hash    text,
      format      text,
      size        integer,
      lastmod_at  integer,
      lastseen_at integer,
      added_at    integer,
      updated_at  integer,
      meta        text,
      audio_meta  text,
      foreign key(store_id) references stores(id)
    );

    create index if not exists files_id_change_idx ON files (id, change_hash);

  `

	_, err = db.Exec(createTableSql)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now().Unix()

	if !strings.HasSuffix(importdir, "/") {
		importdir = importdir + "/"
	}

	root, storeId, err := FindStoreId(importdir)
	if err != nil {
		log.Fatal(err)
	}
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}

	if storeId != "" {
		fmt.Printf("found store id: %s\n", storeId)
	} else {
		fmt.Printf("initialiazing store id in base %s\n", root)
		storeId = uuid.New()
		err := WriteStoreId(root, storeId)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("importing root:%s dir:%s\n", root, importdir)

	_, err = db.Exec(`insert or replace into stores (id) values (?)`, storeId)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	insertFileSql, err := tx.Prepare(`
    insert or replace into files
      (id, store_id, change_hash, track_hash, path, dir_hash, format, 
       size, lastmod_at, lastseen_at, added_at, updated_at, meta, audio_meta) 
    values
      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `)
	if err != nil {
		log.Fatal(err)
	}
	defer insertFileSql.Close()

	updateFileSql, err := tx.Prepare(`
    update files set lastseen_at = ? where id = ?
  `)
	if err != nil {
		log.Fatal(err)
	}
	defer updateFileSql.Close()

	var progresscount uint32 = 0

	WalkForImport(importdir, func(path string, info os.FileInfo) {

		// ensure the path is from the root dir not the base
		path = strings.TrimPrefix(importdir+path, root)

		// fileId represents the path to the file, nothing about the content
		fileId := StringToSHA1(storeId + ":" + path)

		// changeHash does it's best to represent the content of the file
		// an sha1 hash of the file would be best, but it's too slow for importing large libraries
		changeHash := CalculateChangeHash(path, info)

		var existingChangeHash string
		var addedAt int64
		row := tx.QueryRow(`
			select change_hash, added_at from files where id = ?
		`, fileId).Scan(&existingChangeHash, &addedAt)

		exists, err := SqlExists(tx, row)
		if err != nil {
			log.Fatal(err)
		}
		if exists && existingChangeHash == changeHash {
			_, err = updateFileSql.Exec(now, fileId)
			if err != nil {
				log.Fatal(err)
			}
		} else {

			if !exists {
				addedAt = now
			}

			props, audioProps, err := audiotags.Read(root + path)
			if err != nil {
				log.Fatal(err)
			}

			// tries to represent track identity
			trackHash := CalculateTrackHash(props)

			tagdata, err := json.Marshal(props)
			if err != nil {
				log.Fatal(err)
			}
			audiodata, err := json.Marshal(audioProps)
			if err != nil {
				log.Fatal(err)
			}

			format := strings.ToLower(filepath.Ext(path))
			format = format[0 : len(format)-2]

			dirHash := StringToSHA1(filepath.Dir(path))

			_, err = insertFileSql.Exec(fileId, storeId, changeHash, trackHash, path, dirHash, format,
				info.Size(), info.ModTime().Unix(), now, addedAt, now, tagdata, audiodata)
			if err != nil {
				log.Fatal(err)
			}
		}

		num := atomic.AddUint32(&progresscount, 1)

		if num%1000 == 0 {
			fmt.Printf("processed %d entries\n", num)
		}

	})
	tx.Commit()
	fmt.Printf("processed %d entries\n", atomic.LoadUint32(&progresscount))
}

func WalkForImport(root string, fn WalkFunc) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			path := strings.TrimPrefix(path, root)
			lower := strings.ToLower(path)
			if strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".m4a") {
				fn(path, info)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

const META_FILENAME = ".audioplayer.store.id"

func FindStoreId(path string) (string, string, error) {
	current := path
	for current != "" && current != "/" {
		fmt.Printf("checking %s for storeinfo\n", current+META_FILENAME)
		info, err := os.Stat(current + META_FILENAME)
		if err != nil {
			if strings.HasSuffix(current, "/") {
				current = current[0 : len(current)-2]
			}
			current = filepath.Dir(current)
			if !strings.HasSuffix(current, "/") {
				current = current + "/"
			}
		} else if info.IsDir() {
			return "", "", fmt.Errorf("%s should be a file not a directoy", current+META_FILENAME)
		} else {
			bs, err := ioutil.ReadFile(current + META_FILENAME)
			if err != nil {
				return "", "", err
			}
			return current, strings.TrimSpace(string(bs)), nil
		}
	}
	return path, "", nil
}

func WriteStoreId(path string, id string) error {
	return ioutil.WriteFile(path+META_FILENAME, []byte(id+"\n"), 0644)
}

func StringToSHA1(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func CalculateChangeHash(path string, info os.FileInfo) string {
	h := sha1.New()
	lastmod := info.ModTime().Unix() // utc seconds since unix epoch
	size := info.Size()
	// could added in the first 1kb of the file too perhaps
	binary.Write(h, binary.LittleEndian, lastmod)
	binary.Write(h, binary.LittleEndian, size)
	return hex.EncodeToString(h.Sum(nil))
}

func CalculateTrackHash(props map[string]string) string {

	h := sha1.New()

	trackparts := strings.SplitN(props["tracknumber"], "/", 2)
	tracknumber := trackparts[0]

	discparts := strings.SplitN(props["discnumber"], "/", 2)
	discnumber := discparts[0]

	if discnumber == "1" {
		// discnumber 1 is implicitly the case...
		discnumber = ""
	}

	empty := true

	values := []string{
		props["title"],
		props["artist"],
		props["album"],
		props["date"],
		tracknumber,
		discnumber, // will be blank for most things
	}

	for _, s := range values {

		s = processing.ToLowerWithoutPunctuation(s)

		if s != "" {
			empty = false
			io.WriteString(h, s)
		}
	}

	if empty {
		return ""
	} else {
		return hex.EncodeToString(h.Sum(nil))
	}
}

func SqlExists(tx *sql.Tx, err error) (bool, error) {
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}
