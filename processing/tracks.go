package processing

import (
	"encoding/json"
	"github.com/nicksellen/audioplayer/models"
	"log"
	"strconv"
	"strings"
)

func ProcessTracks() {

	db := OpenDB()
	defer db.Close()

	_, err := db.Exec(`

    drop table if exists tracks;

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
	tr := make(map[string][]models.FileEntry)
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
		var entries []models.FileEntry
		if _, ok := tr[track_hash]; ok {
			entries = tr[track_hash]
		} else {
			entries = []models.FileEntry{}
		}
		entries = append(entries, models.FileEntry{Path: path, Meta: meta, AudioMeta: audioMeta})
		tr[track_hash] = entries
	}

	// for each track

	for th, entries := range tr {

		m := make(map[string]string)

		m["track_hash"] = th

		// ...and each file that is of the track

		for _, e := range entries {

			propertyMerge(m, e.Meta, "albumartist")
			propertyMerge(m, e.Meta, "artist")
			propertyMerge(m, e.Meta, "album")
			propertyMerge(m, e.Meta, "title")
			propertyMerge(m, e.Meta, "composer")
			propertyMerge(m, e.Meta, "compilation")
			propertyMerge(m, e.Meta, "label")

			e.Meta["year"] = e.Meta["originaldate"]
			if e.Meta["year"] == "" {
				e.Meta["year"] = e.Meta["date"]
			}

			e.Meta["year"] = YEAR_RE.FindString(e.Meta["year"])

			propertyMergeWithPropertyConflictHandler(m, e.Meta, "year", func(a string, b string) string {
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

			propertyMergeWithPropertyConflictHandler(m, e.Meta, "tracknumber", func(a string, b string) string {
				if strings.Contains(a, "/") {
					return a
				}
				return b
			})

			propertyMergeWithPropertyConflictHandler(m, e.Meta, "discnumber", func(a string, b string) string {
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
		}

		insertTrackSql.Exec(
			nullString(m["track_hash"]),
			nullString(m["title"]),
			nullString(m["artist"]),
			nullString(m["album"]),
			nullString(m["albumartist"]),
			nullString(m["composer"]),
			nullString(m["label"]),
			nullString(m["albumsort"]),
			nullString(m["albumartistsort"]),
			nullString(m["artistsort"]),
			nullString(m["titlesort"]),
			nullString(m["compilation"]),
			nullInt64(m["year"]),
			nullInt64(m["tracknumber"]),
			nullInt64(m["totaltracks"]),
			nullInt64(m["discnumber"]),
			nullInt64(m["totaldiscs"]))
	}

	tx.Commit()
}
