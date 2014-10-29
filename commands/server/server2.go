package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nicksellen/audioplayer/models"
	"log"
	"net/http"
)

type AlbumDetails struct {
	Name   string       `json:"name"`
	Tracks []AlbumTrack `json:"tracks"`
}

type AlbumTrack struct {
	Id      int64              `json:"id"`
	Title   string             `json:"name"`
	Artist  string             `json:"artist"`
	Pos     int                `json:"pos"`
	Sources []AlbumTrackSource `json:"sources"`
}

type AlbumTrackSource struct {
	Url         string `json:"url"`
	Format      string `json:"format"`
	ContentType string `json:"contentType"`
}

type TrackDetails struct {
	Id      int64    `json:"id"`
	Title   string   `json:"name"`
	Artist  string   `json:"artist"`
	Pos     int      `json:"pos"`
	Formats []string `json:"formats"`
}

type AlbumListResult struct {
	Albums []models.DBAlbum `json:"albums"`
}

type TrackListResult struct {
	Tracks []models.DBTrack `json:"tracks"`
}

func Server2() {

	db, err := sql.Open("sqlite3", "./db.v2.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	sqlxdb := sqlx.NewDb(db, "sqlite3")
	defer db.Close()

	router := mux.NewRouter()

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/index.html")
	}

	// all the pages react can handle, just give it the index.html
	indexRoutes := []string{
		"/",
		"/albums",
		"/albums/{name:.*}",
		"/player",
	}
	for _, r := range indexRoutes {
		router.HandleFunc(r, serveIndex)
	}

	router.Handle("/api/tracks", NewTrackListHandler(sqlxdb))

	router.Handle("/api/albums", NewAlbumListHandler(sqlxdb))
	router.Handle("/api/albums/{name:.+}", NewAlbumGetHandler(db, "name"))
	router.Handle("/audio/{id:[0-9]+}.{format}", NewAudioHandler(db, "id")).Methods("GET")

	http.Handle("/", router)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	port := 8081
	fmt.Printf("listening on %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func NewTrackListHandler(db *sqlx.DB) *TrackListHandler {
	return &TrackListHandler{db: db}
}

func NewAudioHandler(db *sql.DB, idParam string) *AudioHandler {
	return &AudioHandler{db: db, idParam: idParam}
}

func NewAlbumGetHandler(db *sql.DB, nameParam string) *AlbumGetHandler {
	return &AlbumGetHandler{db: db, nameParam: nameParam}
}

func NewAlbumListHandler(db *sqlx.DB) *AlbumListHandler {
	return &AlbumListHandler{db: db}
}

type TrackListHandler struct {
	db *sqlx.DB
}

func (h *TrackListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tracks := []models.DBTrack{}
	err := h.db.Select(&tracks, "select * from tracks")
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.Marshal(&TrackListResult{Tracks: tracks})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(b)
}

type AlbumListHandler struct {
	db *sqlx.DB
}

func (h *AlbumListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	albums := []models.DBAlbum{}
	err := h.db.Select(&albums, `
    select 
      id,
      album, 
      albumartist,
      artists,  
      trackcount,
      totaltracks,
      discnumber,
      totaldiscs,
      year,
      toyear
    from albums
  `)
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.Marshal(&AlbumListResult{Albums: albums})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(b)
}

type AlbumGetHandler struct {
	db        *sql.DB
	nameParam string
}

func (h *AlbumGetHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(req)
	name := vars[h.nameParam]
	fmt.Printf("looking for album: %s\n", name)
	rows, err := h.db.Query(`
      select t.id, t.title, t.album, t.artist, t.track, f.format, f.path
      from tracks t
      join files f on f.track_id = t.id
      where t.album = ?
      order by t.track
    `, name)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	trackMap := make(map[int64]AlbumTrack)
	trackIds := []int64{}
	for rows.Next() {
		var id int64
		var title string
		var album string
		var artist string
		var track int
		var format string
		var path string
		rows.Scan(&id, &title, &album, &artist, &track, &format, &path)
		var t AlbumTrack
		if _, ok := trackMap[id]; ok {
			t = trackMap[id]
		} else {
			t = AlbumTrack{Id: id, Title: title, Artist: artist, Pos: track, Sources: []AlbumTrackSource{}}
			trackIds = append(trackIds, id)
		}
		t.Sources = append(t.Sources, AlbumTrackSource{
			Format:      format,
			ContentType: FormatToContentType(format),
			Url:         fmt.Sprintf("/audio/%d.%s", id, format),
		})
		trackMap[id] = t
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	tracks := []AlbumTrack{}
	for _, id := range trackIds {
		tracks = append(tracks, trackMap[id])
	}
	albumDetails := AlbumDetails{Name: name, Tracks: tracks}
	b, err := json.Marshal(albumDetails)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(b)
}
