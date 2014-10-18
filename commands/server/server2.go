package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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

type Track struct {
	Id          int    `json:"id,omitempty"`
	TrackHash   string `json:"-" db:"track_hash"`
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"albumartist,omitempty"`

	Composer    string `json:"composer,omitempty"`
	Label       string `json:"label,omitempty"`
	Compilation string `json:"compilation,omitempty"`
	Year        int    `json:"year,omitempty"`

	TrackNumber int `json:"tracknumber,omitempty"`
	TotalTracks int `json:"totaltracks,omitempty"`
	DiscNumber  int `json:"discnumber,omitempty"`
	TotalDiscs  int `json:"totaldiscs,omitempty"`

	AlbumSort       string `json:"albumsort,omitempty"`
	AlbumArtistSort string `json:"albumartistsort,omitempty"`
	ArtistSort      string `json:"artistsort,omitempty"`
	TitleSort       string `json:"titlesort,omitempty"`
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

	router.Handle("/api/albums", NewAlbumListHandler(db))
	router.Handle("/api/albums/{name:.+}", NewAlbumGetHandler(db, "name"))
	router.Handle("/audio/{id:[0-9]+}.{format}", NewAudioHandler(db, "id")).Methods("GET")

	http.Handle("/", router)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func FormatToContentType(format string) string {
	format = strings.ToLower(format)
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "m4a", "mp4":
		return "audio/mp4"
	default:
		log.Fatalf("unknown format %s", format)
	}
	return ""
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

func NewAlbumListHandler(db *sql.DB) *AlbumListHandler {
	return &AlbumListHandler{db: db}
}

type TrackListHandler struct {
	db *sqlx.DB
}

type TrackListResult struct {
	Tracks []Track `json:"tracks"`
}

func (h *TrackListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tracks := []Track{}
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

type AudioHandler struct {
	db      *sql.DB
	idParam string
}

func (h *AudioHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseInt(vars[h.idParam], 10, 64)
	format := vars["format"]
	if err != nil {
		log.Fatal(err)
	}
	row := h.db.QueryRow(`
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
	w.Header().Set("Content-Type", FormatToContentType(path[len(path)-3:]))
	http.ServeFile(w, req, store+path)
}

type AlbumListHandler struct {
	db *sql.DB
}

func (h *AlbumListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	albums := []Album{}
	rows, err := h.db.Query(`
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
}

type AlbumGetHandler struct {
	db        *sql.DB
	nameParam string
}

func (h *AlbumGetHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(req)
	name := vars[h.nameParam]
	fmt.Printf("looking album: %s\n", name)
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
