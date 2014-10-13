package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	Name    string `json:"name"`
	Artists string `json:"artists"`
}

type AlbumDetails struct {
	Name   string       `json:"name"`
	Tracks []AlbumTrack `json:"tracks"`
}

type AlbumTrack struct {
	Id      int64    `json:"id"`
	Title   string   `json:"name"`
	Artist  string   `json:"artist"`
	Pos     int      `json:"pos"`
	Formats []string `json:"formats"`
}

type TrackDetails struct {
	Id      int64    `json:"id"`
	Title   string   `json:"name"`
	Artist  string   `json:"artist"`
	Pos     int      `json:"pos"`
	Formats []string `json:"formats"`
}

func Server() {

	db, err := sql.Open("sqlite3", "./db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := mux.NewRouter()

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/index.html")
	}

	router.HandleFunc("/", serveIndex)
	router.HandleFunc("/albums", serveIndex)
	router.HandleFunc("/albums/{name:.+}", serveIndex)

	router.HandleFunc("/api/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		artists := []Artist{}
		rows, err := db.Query(`
			select artist from tracks
			group by artist
			order by artist
		`)
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

	router.HandleFunc("/api/albums", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		albums := []Album{}
		rows, err := db.Query(`
			select 
				album, 
				group_concat(distinct artist) as artists 
			from tracks 
			where album is not null and album != ''
			group by album 
			order by lower(album)
		`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var album string
			var artists string
			rows.Scan(&album, &artists)
			albums = append(albums, Album{Name: album, Artists: artists})
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

	router.HandleFunc("/api/albums/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		tracks := []AlbumTrack{}
		name := vars["name"]
		fmt.Printf("looking album: %s\n", name)
		rows, err := db.Query(`
			select t.id, t.title, t.album, t.artist, t.track, 
						 group_concat(distinct f.format) as formats
			from tracks t
			join files f on f.track_id = t.id
			where t.album = ?
			group by t.id
			order by t.track
		`, name)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var title string
			var album string
			var artist string
			var track int
			var formats string
			rows.Scan(&id, &title, &album, &artist, &track, &formats)
			tracks = append(tracks, AlbumTrack{Id: id, Title: title, Artist: artist, Pos: track, Formats: strings.Split(formats, ",")})
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		albumDetails := AlbumDetails{Name: name, Tracks: tracks}
		b, err := json.Marshal(albumDetails)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(b)
	})

	router.HandleFunc("/api/tracks/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		row := db.QueryRow(`
				select title, album, artist, track
				from tracks
				where id = ?
			`, id)
		var title string
		var album string
		var artist string
		var track int
		err = row.Scan(&title, &album, &artist, &track)
		if err != nil {
			log.Fatal(err)
		}
		tr := TrackDetails{Id: id, Title: title, Artist: artist, Pos: track}
		b, err := json.Marshal(tr)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(b)
	})

	router.HandleFunc("/audio/{id:[0-9]+}.{format}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		format := vars["format"]
		if err != nil {
			log.Fatal(err)
		}
		row := db.QueryRow(`
			select t.title, f.path, s.name
			from tracks t
			left join files f on f.track_id = t.id
			left join stores s on f.store_id = s.id
			where t.id = ? and f.format = ?
			limit 1
		`, id, format)
		var title string
		var path string
		var store string
		err = row.Scan(&title, &path, &store)
		if err != nil {
			log.Fatal(err)
		}

		filename := store + path
		lower := strings.ToLower(filename)

		if strings.HasSuffix(lower, ".mp3") {
			w.Header().Set("Content-Type", "audio/mpeg")
		} else if strings.HasSuffix(lower, ".m4a") {
			w.Header().Set("Content-Type", "audio/mp4")
		} else {
			log.Fatal("not mp4 or m4a")
		}

		http.ServeFile(w, r, filename)
	})

	http.Handle("/", router)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
