package commands

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

type ArtistList struct {
	Artists []Artist `json:"artists"`
}

type Artist struct {
	Name string `json:"name"`
}

type AlbumList struct {
	Albums []Album `json:"albums"`
}

type Album struct {
	Name string `json:"name"`
}

func Server() {

	db, err := sql.Open("sqlite3", "./db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	http.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		artists := []Artist{}
		rows, err := db.Query("select artist from tracks group by artist order by artist")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			rows.Scan(&name)
			artists = append(artists, Artist{Name: name})
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		artistList := ArtistList{Artists: artists}
		b, err := json.Marshal(artistList)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(b)
	})

	http.HandleFunc("/albums", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		albums := []Album{}
		rows, err := db.Query("select album from tracks group by album order by album")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var album string
			rows.Scan(&album)
			albums = append(albums, Album{Name: album})
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		albumList := AlbumList{Albums: albums}
		b, err := json.Marshal(albumList)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(b)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
