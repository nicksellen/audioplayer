package commands

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func Show() {

	db, err := sql.Open("sqlite3", "./db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`
      select 
        t.id, 
        t.artist, 
        t.album, 
        t.title, 
        t.track, 
        f.path,
        s.name
      from tracks t
      join files f 
        on f.track_id = t.id
      join stores s
        on f.store_id = s.id
    `)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var trackId int
		var artist string
		var album string
		var title string
		var track int
		var path string
		var store string
		rows.Scan(&trackId, &artist, &album, &title, &track, &path, &store)
		fmt.Printf("artist : %s\nalbum  : %s\ntitle  : %s\ntrack  : %d\npath   : %s\nstore  : %s\n\n",
			artist, album, title, track, path, store)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	rows.Close()
}
