package commands

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wtolson/go-taglib"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	//"sync"
	"sync/atomic"
)

type WalkFunc2 func(path string)

func Index(root string) {

	runtime.GOMAXPROCS(1)

	fmt.Printf("importing from %s\n", root)

	//os.Remove("./db")

	db, err := sql.Open("sqlite3", "./db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSql := `

    create table if not exists tracks (
      id integer primary key,
      artist text, 
      album text, 
      title text, 
      track integer
    );

    create table if not exists stores (
      id integer primary key,
      name text
    );

    create table if not exists files (
      id integer primary key,
      store_id integer,
      track_id integer,
      path text,
      format text,
      foreign key(store_id) references stores(id),
      foreign key(track_id) references tracks(id)
    );

  `

	_, err = db.Exec(createTableSql)
	if err != nil {
		log.Fatal(err)
	}

	var storeId int64
	err = db.QueryRow(`select id from stores where name = ?`, root).Scan(&storeId)

	switch {
	case err == sql.ErrNoRows:
		result, err := db.Exec("insert into stores (name) values (?)", root)
		if err != nil {
			log.Fatal(err)
		}
		storeId, err = result.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}
	case err != nil:
		log.Fatal(err)
	default:
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	insertTrackSql, err := tx.Prepare(`
		insert into tracks (artist, album, title, track) values (?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer insertTrackSql.Close()

	insertFileSql, err := tx.Prepare(`
		insert into files (store_id, track_id, path, format) values (?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer insertFileSql.Close()

	var filecount uint32 = 0
	var processedcount uint32 = 0

	//var wg sync.WaitGroup

	Walk(root, func(path string) {
		//wg.Add(1)
		atomic.AddUint32(&filecount, 1)
		//go func() {
		//defer wg.Done()
		track, err := taglib.Read(root + path)
		atomic.AddUint32(&processedcount, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %s\n", path, err)
		} else {
			num := atomic.LoadUint32(&processedcount)
			if num%1000 == 0 {
				fmt.Fprintf(os.Stderr, "%d/%d\n", num, atomic.LoadUint32(&filecount))
			}

			var trackId int64
			err := tx.QueryRow(`
					select id from tracks
					where
						artist = ? and
						album = ? and
						title = ? and
						track = ?
				`, track.Artist(), track.Album(), track.Title(), track.Track()).Scan(&trackId)

			switch {
			case err == sql.ErrNoRows:
				result, err := insertTrackSql.Exec(track.Artist(), track.Album(), track.Title(), track.Track())
				if err != nil {
					log.Fatal(err)
				}
				trackId, err = result.LastInsertId()
				if err != nil {
					log.Fatal(err)
				}
			case err != nil:
				log.Fatal(err)
			default:
			}
			format := strings.ToLower(filepath.Ext(path))

			var fileId int64
			err = tx.QueryRow(`
					select id from files
					where
						store_id = ? and
						track_id = ? and
						path = ? and
						format = ?
				`, storeId, trackId, path, format).Scan(&fileId)

			switch {
			case err == sql.ErrNoRows:
				_, err = insertFileSql.Exec(storeId, trackId, path, format[1:])
				if err != nil {
					log.Fatal(err)
				}
			case err != nil:
				log.Fatal(err)
			default:
			}

			track.Close()
		}
		//}()
	})
	//wg.Wait()
	tx.Commit()
	fmt.Fprintf(os.Stderr, "%d/%d\n", atomic.LoadUint32(&processedcount), atomic.LoadUint32(&filecount))
}

func Walk(root string, fn WalkFunc2) {
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
