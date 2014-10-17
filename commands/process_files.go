package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/arbovm/levenshtein"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"regexp"
	"strings"
)

type FileEntry struct {
	path      string
	meta      map[string]string
	audioMeta map[string]int
}

func ProcessFiles() {

	db, err := sql.Open("sqlite3", "./db.v2.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`

    create table if not exists tracks (
      id integer primary key,
      track_hash  text,
      title       text,
      artist      text,
      album       text,
      albumartist text,
      year        int,
      tracknumber int,
      trackcount  int
    );

  `)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(`
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
		var trackHash string
		var metaStr string
		var audioMetaStr string
		rows.Scan(&path, &trackHash, &metaStr, &audioMetaStr)

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
		if _, ok := tr[trackHash]; ok {
			entries = tr[trackHash]
		} else {
			entries = []FileEntry{}
		}
		entries = append(entries, FileEntry{path: path, meta: meta, audioMeta: audioMeta})
		tr[trackHash] = entries
	}

	// for each track
	for th, entries := range tr {
		fmt.Printf("------------ %s %d--------------\n", th, len(entries))

		final := make(map[string]string)

		// ...and each file that is of the track
		for _, e := range entries {
			fmt.Printf("  - %s\n", e.path)

			propertyMerge(final, "albumartist", e.meta)
			propertyMerge(final, "artist", e.meta)
			propertyMerge(final, "album", e.meta)
			propertyMerge(final, "title", e.meta)

			// the meta from that file
			for k, v := range e.meta {
				fmt.Printf("    %s : %s\n", k, v)
			}
		}

		fmt.Printf("  - final:\n")

		for k, v := range final {
			fmt.Printf("    %s : %s\n", k, v)
		}
	}

}

var CAP = regexp.MustCompile("[A-Z]")

func propertyMerge(dest map[string]string, k string, incoming map[string]string) {
	existing := strings.TrimSpace(dest[k])
	v := strings.TrimSpace(incoming[k])

	if existing == v {
		// easy
	} else if existing == "" || v == "" {
		if v != "" {
			dest[k] = v
		}
	} else {
		resolved := false
		existingLower := strings.ToLower(existing)
		vLower := strings.ToLower(v)
		if existingLower == vLower {
			// the only difference is capitalization, use the one with the most capitalization
			if CountOccurances(v, CAP) > CountOccurances(existing, CAP) {
				dest[k] = v
			}
			resolved = true
		} else {
			dist := levenshtein.Distance(existing, v)
			if dist < 3 {
				resolved = true
				dest[k] = v
			}
		}
		if !resolved {
			fmt.Printf("conflict: %s [%s] vs [%s]\n", k, existing, v)
		}
	}
}

func CountOccurances(s string, re *regexp.Regexp) int {
	return len(re.FindAllString(s, -1))
}
