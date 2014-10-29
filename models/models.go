package models

import (
	"database/sql"
	"encoding/json"
)

type FileEntry struct {
	Path      string
	DirHash   string
	Meta      map[string]string
	AudioMeta map[string]int
}

type DBAlbum struct {
	Id          int64
	Album       *sql.NullString `db:"album"`
	AlbumArtist *sql.NullString
	Artists     *sql.NullString
	TotalTracks *sql.NullInt64
	DiscNumber  *sql.NullInt64
	TotalDiscs  *sql.NullInt64
	Year        *sql.NullInt64
	ToYear      *sql.NullInt64
	Incomplete  *sql.NullBool
}

func (a *DBAlbum) FixNulls() *DBAlbum {
	return &DBAlbum{
		0,
		ensureNullString(a.Album),
		ensureNullString(a.AlbumArtist),
		ensureNullString(a.Artists),
		ensureNullInt64(a.TotalTracks),
		ensureNullInt64(a.DiscNumber),
		ensureNullInt64(a.TotalDiscs),
		ensureNullInt64(a.Year),
		ensureNullInt64(a.ToYear),
		ensureNullBool(a.Incomplete),
	}
}

func (a *DBAlbum) ToAlbum() *Album {
	return &Album{
		a.Id,
		maybeNullString(a.Album),
		maybeNullString(a.AlbumArtist),
		maybeNullString(a.Artists),
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
	Id          int64  `json:"id"`
	Album       string `json:"name,omitempty"`
	AlbumArtist string `json:"albumartists,omitempty"`
	Artists     string `json:"artists,omitempty"`
	TotalTracks int64  `json:"totaltracks,omitempty"`
	DiscNumber  int64  `json:"discnumber,omitempty"`
	TotalDiscs  int64  `json:"totaldiscs,omitempty"`
	Year        int64  `json:"year,omitempty"`
	ToYear      int64  `json:"toyear,omitempty"`
	Incomplete  bool   `json:"incomplete,omitempty"`
}

func (a *Album) ToDBAlbum() *DBAlbum {
	return &DBAlbum{}
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

func ensureNullString(s *sql.NullString) *sql.NullString {
	if s != nil {
		return s
	}
	return &sql.NullString{"", false}
}

func ensureNullInt64(i *sql.NullInt64) *sql.NullInt64 {
	if i != nil {
		return i
	}
	return &sql.NullInt64{0, false}
}

func ensureNullBool(b *sql.NullBool) *sql.NullBool {
	if b != nil {
		return b
	}
	return &sql.NullBool{false, false}
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
