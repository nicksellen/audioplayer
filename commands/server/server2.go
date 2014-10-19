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
	Albums []DBAlbum `json:"albums"`
}

type DBAlbum struct {
	Name        *sql.NullString `db:"album"`
	AlbumArtist *sql.NullString
	Artists     *sql.NullString
	TrackCount  *sql.NullInt64
	TotalTracks *sql.NullInt64
	DiscNumber  *sql.NullInt64
	TotalDiscs  *sql.NullInt64
	Year        *sql.NullInt64
	ToYear      *sql.NullInt64
	Incomplete  *sql.NullBool
}

func (a *DBAlbum) ToAlbum() *Album {
	return &Album{
		maybeNullString(a.Name),
		maybeNullString(a.AlbumArtist),
		maybeNullString(a.Artists),
		maybeNullInt64(a.TrackCount),
		maybeNullInt64(a.TotalTracks),
		maybeNullInt64(a.DiscNumber),
		maybeNullInt64(a.TotalDiscs),
		maybeNullInt64(a.Year),
		maybeNullInt64(a.ToYear),
		maybeNullBool(a.Incomplete),
	}
}

func (a *DBAlbum) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.ToAlbum())
}

type Album struct {
	Name        string `json:"name,omitempty"`
	AlbumArtist string `json:"albumartists,omitempty"`
	Artists     string `json:"artists,omitempty"`
	TrackCount  int64  `json:"trackcount,omitempty"`
	TotalTracks int64  `json:"totaltracks,omitempty"`
	DiscNumber  int64  `json:"discnumber,omitempty"`
	TotalDiscs  int64  `json:"totaldiscs,omitempty"`
	Year        int64  `json:"year,omitempty"`
	ToYear      int64  `json:"toyear,omitempty"`
	Incomplete  bool   `json:"incomplete,omitempty"`
}

type TrackListResult struct {
	Tracks []DBTrack `json:"tracks"`
}

type DBTrack struct {
	Id          int64
	TrackHash   *sql.NullString `db:"track_hash"`
	Title       *sql.NullString
	Artist      *sql.NullString
	Album       *sql.NullString
	AlbumArtist *sql.NullString

	Composer    *sql.NullString
	Label       *sql.NullString
	Compilation *sql.NullString
	Year        *sql.NullInt64

	TrackNumber *sql.NullInt64
	TotalTracks *sql.NullInt64
	DiscNumber  *sql.NullInt64
	TotalDiscs  *sql.NullInt64

	AlbumSort       *sql.NullString
	AlbumArtistSort *sql.NullString
	ArtistSort      *sql.NullString
	TitleSort       *sql.NullString
}

func (t *DBTrack) ToTrack() *Track {
	return &Track{
		t.Id,
		maybeNullString(t.TrackHash),
		maybeNullString(t.Title),
		maybeNullString(t.Artist),
		maybeNullString(t.Album),
		maybeNullString(t.AlbumArtist),
		maybeNullString(t.Composer),
		maybeNullString(t.Label),
		maybeNullString(t.Compilation),
		maybeNullInt64(t.Year),
		maybeNullInt64(t.TrackNumber),
		maybeNullInt64(t.TotalTracks),
		maybeNullInt64(t.DiscNumber),
		maybeNullInt64(t.TotalDiscs),
		maybeNullString(t.AlbumSort),
		maybeNullString(t.AlbumArtistSort),
		maybeNullString(t.ArtistSort),
		maybeNullString(t.TitleSort),
	}
}

func (t *DBTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ToTrack())
}

func maybeNullString(s *sql.NullString) string {
	if s != nil && s.Valid {
		return s.String
	}
	return ""
}

func maybeNullInt64(i *sql.NullInt64) int64 {
	if i != nil && i.Valid {
		return i.Int64
	}
	return 0
}

func maybeNullBool(b *sql.NullBool) bool {
	if b != nil && b.Valid {
		return b.Bool
	}
	return false
}

type Track struct {
	Id          int64  `json:"id,omitempty"`
	TrackHash   string `json:"-"`
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"albumartist,omitempty"`

	Composer    string `json:"composer,omitempty"`
	Label       string `json:"label,omitempty"`
	Compilation string `json:"compilation,omitempty"`
	Year        int64  `json:"year,omitempty"`

	TrackNumber int64 `json:"tracknumber,omitempty"`
	TotalTracks int64 `json:"totaltracks,omitempty"`
	DiscNumber  int64 `json:"discnumber,omitempty"`
	TotalDiscs  int64 `json:"totaldiscs,omitempty"`

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

	router.Handle("/api/albums", NewAlbumListHandler(sqlxdb))
	router.Handle("/api/albums/{name:.+}", NewAlbumGetHandler(db, "name"))
	router.Handle("/audio/{id:[0-9]+}.{format}", NewAudioHandler(db, "id")).Methods("GET")

	http.Handle("/", router)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	log.Fatal(http.ListenAndServe(":8081", nil))
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
	tracks := []DBTrack{}
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
	albums := []DBAlbum{}
	err := h.db.Select(&albums, `
  
    -- woah looks hairy eh? take a deep breath...

    select
      coalesce(albumartist, 'Various Artists') as albumartist,
      nullif(artists, albumartist) as artists,
      album,

      sum(trackcount) as trackcount,
      min(totaltracks) as totaltracks,
      min(discnumber) as discnumber,
      min(totaldiscs) as totaldiscs,
      min(year) as year,
      max(toyear) as toyear,

      -- very basic metric of incompleteness
      min(mintracknumber) != 1 as incomplete 

    from (

      select 

        -- we want the albumartist
        -- we might be able to generate it
        -- if all the tracks have the same artist
        coalesce(
          t.albumartist, 
          (case count(distinct t.artist)
            when 1 then t.artist
            else null -- meaning various
          end)
        ) as albumartist,

        -- include a mention all the other artists
        group_concat(distinct nullif(t.artist, t.albumartist)) as artists,

        t.album, 

        min(t.discnumber) as discnumber,
        min(t.totaldiscs) as totaldiscs,

        min(t.totaltracks) as totaltracks,
        count(distinct t.id) as trackcount,

        min(year) year,
        nullif(max(year), min(year)) toyear,

        min(t.tracknumber) as mintracknumber

      from tracks t 
      join files f on t.track_hash = f.track_hash

      where 
        t.album is not null
        and t.artist is not null
        and t.title is not null

        -- a few cases I wanted to peek at manually

        --and t.album = 'One'
        --and t.album = 'Good Rain'
        --and t.album = 'Lontano'
        --and t.album like 'Always Outnumbered%'
        --and t.album like 'Angola - The greatest%'

      group by t.album, t.albumartist

    ) q

    --where trackcount = totaltracks

    -- regroup as our inner query may have successfully
    -- found an albumartist from the chaos and it might be able
    -- to be paired up with it's siblings
    group by album, albumartist

    --order by albumartist, album
    order by album

    ;


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
