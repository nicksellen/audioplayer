package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/arbovm/levenshtein"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type FileEntry struct {
	path      string
	dirHash   string
	meta      map[string]string
	audioMeta map[string]int
}

type Track struct {
	Id          int
	track_hash  string
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Year        int
	TrackNumber int
	TrackCount  int
	DiscNumber  int
	DiscCount   int
}

type ConflictHandler func(string, string) string

func ProcessFiles() {

	db, err := sql.Open("sqlite3", "./db.v2.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`

    create table if not exists tracks (

      id integer primary key,

      track_hash       text,

      title           text,
      artist          text,
      album           text,

      albumartist     text,
      composer        text,
      label           text,

      albumsort       text,
      albumartistsort text,
      artistsort      text,
      titlesort       text,

      compilation     text,

      year            int,

      tracknumber     int,
      totaltracks     int,
      discnumber      int,
      totaldiscs      int

    );
  `)

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	insertTrackSql, err := tx.Prepare(`
    insert or replace into tracks
      (track_hash,
       title,
       artist,
       album,
       albumartist,
       composer,
       label,
       albumsort,
       albumartistsort,
       artistsort,
       titlesort,
       compilation,
       year,
       tracknumber,
       totaltracks,
       discnumber,
       totaldiscs
      ) values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
  `)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := tx.Query(`
    select path, track_hash, meta, audio_meta from files where track_hash != ''
  `)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tr := make(map[string][]FileEntry)
	for rows.Next() {
		var path string
		var track_hash string
		var metaStr string
		var audioMetaStr string
		rows.Scan(&path, &track_hash, &metaStr, &audioMetaStr)

		meta := make(map[string]string)
		err = json.Unmarshal([]byte(metaStr), &meta)
		if err != nil {
			log.Fatal(err)
		}
		audioMeta := make(map[string]int)
		err := json.Unmarshal([]byte(audioMetaStr), &audioMeta)
		if err != nil {
			log.Fatal(err)
		}
		var entries []FileEntry
		if _, ok := tr[track_hash]; ok {
			entries = tr[track_hash]
		} else {
			entries = []FileEntry{}
		}
		entries = append(entries, FileEntry{path: path, meta: meta, audioMeta: audioMeta})
		tr[track_hash] = entries
	}

	// for each track
	for th, entries := range tr {
		//fmt.Printf("------------ %s %d--------------\n", th, len(entries))

		m := make(map[string]string)

		m["track_hash"] = th

		// ...and each file that is of the track
		for _, e := range entries {

			propertyMerge(m, e.meta, "albumartist")
			propertyMerge(m, e.meta, "artist")
			propertyMerge(m, e.meta, "album")
			propertyMerge(m, e.meta, "title")
			propertyMerge(m, e.meta, "composer")
			propertyMerge(m, e.meta, "compilation")
			propertyMerge(m, e.meta, "label")

			e.meta["year"] = e.meta["originaldate"]
			if e.meta["year"] == "" {
				e.meta["year"] = e.meta["date"]
			}

			e.meta["year"] = YEAR_RE.FindString(e.meta["year"])

			propertyMergeWithConflictHandler(m, e.meta, "year", func(a string, b string) string {
				if a == "" {
					return b
				} else if b == "" {
					return a
				} else {
					// just return the smallest one
					ai, err := strconv.Atoi(a)
					if err != nil {
						ai = 0
					}
					bi, err := strconv.Atoi(b)
					if err != nil {
						bi = 0
					}
					if ai < bi {
						return a
					}
					return b
				}
			})

			propertyMergeWithConflictHandler(m, e.meta, "tracknumber", func(a string, b string) string {
				if strings.Contains(a, "/") {
					return a
				}
				return b
			})

			propertyMergeWithConflictHandler(m, e.meta, "discnumber", func(a string, b string) string {
				if strings.Contains(a, "/") {
					return a
				}
				return b
			})

			var s string
			var p []string
			s = m["tracknumber"]
			p = strings.SplitN(s, "/", 2)
			if p[0] != "" {
				m["tracknumber"] = p[0]
			}
			if len(p) > 1 {
				m["totaltracks"] = p[1]
			}

			s = m["discnumber"]
			p = strings.SplitN(s, "/", 2)
			if p[0] != "" {
				m["discnumber"] = p[0]
			}
			if len(p) > 1 {
				m["totaldiscs"] = p[1]
			}

			enforceNum(m, "tracknumber")
			enforceNum(m, "totaltracks")
			enforceNum(m, "discnumber")
			enforceNum(m, "totaldiscs")

			/*
				// the meta from that file
				for k, v := range e.meta {
					fmt.Printf("    %s : %s\n", k, v)
				}
			*/
		}

		/*

			fmt.Printf("  - final:\n")

			for k, v := range m {
				if v != "" {
					fmt.Printf("    %s : %s\n", k, v)
				}
			}
		*/

		insertTrackSql.Exec(
			m["track_hash"],
			m["title"],
			m["artist"],
			m["album"],
			m["albumartist"],
			m["composer"],
			m["label"],
			m["albumsort"],
			m["albumartistsort"],
			m["artistsort"],
			m["titlesort"],
			m["compilation"],
			num(m["year"]),
			num(m["tracknumber"]),
			num(m["totaltracks"]),
			num(m["discnumber"]),
			num(m["totaldiscs"]))
	}

	tx.Commit()

}
func num(val string) int {
	if val != "" {
		n, err := strconv.Atoi(val)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}
	// this would ideally be nil
	// the db is happy with that
	// but loading at the other end with sqlx doesn't like it
	// TODO: I think I need to stop using sqlx for this....
	return 0
}

func enforceNum(m map[string]string, key string) {
	if val, ok := m[key]; ok {
		m[key] = NUM_RE.FindString(val)
	}
}

var YEAR_RE = regexp.MustCompile("[0-9]{4}")
var NUM_RE = regexp.MustCompile("[0-9]+")
var CAP = regexp.MustCompile("[A-Z]")

func propertyMergeWithConflictHandler(m1 map[string]string, m2 map[string]string, k string, resolver ConflictHandler) {
	a := strings.TrimSpace(m1[k])
	b := strings.TrimSpace(m2[k])
	if a == b {
		// easy
	} else if b == "" {
		// nothing to do m1 already contains a
	} else if b != "" {
		m1[k] = b
	} else {
		m1[k] = resolver(a, b)
	}
}

func propertyMerge(m1 map[string]string, m2 map[string]string, k string) {
	propertyMergeWithConflictHandler(m1, m2, k, func(a string, b string) string {
		if EqualsCaseInsenstive(a, b) {
			// the only difference is capitalization, use the one with the most capitalization
			if CountOccurances(b, CAP) > CountOccurances(a, CAP) {
				return b
			}
			return a
		} else {
			dist := levenshtein.Distance(a, b)
			if dist < 3 {
				return b
			}
		}
		fmt.Printf("conflict: %s [%s] vs [%s]\n", k, a, b)
		return a
	})
}

func CountOccurances(s string, re *regexp.Regexp) int {
	return len(re.FindAllString(s, -1))
}

func EqualsCaseInsenstive(a string, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}
