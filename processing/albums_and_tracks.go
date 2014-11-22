package processing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/nicksellen/audioplayer/models"
	"log"
)

func ProcessTracksAndAlbums() {

	db := OpenDB()
	dbx := sqlx.NewDb(db, "sqlite3")
	defer db.Close()

	dbx.MustExec(`

    drop table if exists albums;

    create table if not exists albums (

      id integer primary key,

      album           text,
      artist          text,

      compilation     text,

      year            int,
      toyear          int,

      totaltracks     int,

      discnumber      int,
      totaldiscs      int

    );

  `)

	dbx.MustExec(`

    drop table if exists album_files;

    create table if not exists album_files (
      album_id int,
      track_id text
    );

    create unique index if not exists album_files_uniq_idx
    on album_files (album_id, track_id);

  `)

	tx := dbx.MustBegin()

	insertAlbum, err := tx.PrepareNamed(`
    insert into albums (
      album, artist
    ) values (
      :album, :artist
    )
  `)
	if err != nil {
		log.Fatal(err)
	}

	insertAlbumFile, err := tx.Prepare(`
    insert into album_files (
      album_id, track_id
    ) values (
      ?, ?
    )
  `)
	if err != nil {
		log.Fatal(err)
	}

	// go over all files

	rows, err := tx.Query(`
    select id, path, track_hash, meta, audio_meta from files where track_hash != ''
  `)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var fileid string
		var path string
		var track_hash string
		var metaStr string
		var audioMetaStr string
		rows.Scan(&fileid, &path, &track_hash, &metaStr, &audioMetaStr)
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
		fmt.Printf("%s\n  %s\n  %s\n  %s\n\n",
			meta["title"], meta["album"], meta["albumartist"], meta["artist"])

		albumartist := meta["albumartist"]
		if albumartist == "" {
			albumartist = meta["artist"]
		}
		album, err := LookupAlbum(tx, meta["album"], albumartist)
		if err != nil {
			log.Fatal(err)
		}

		var albumId int64

		if album == nil {
			album = models.NewDBAlbum(meta["album"], albumartist)
			res, err := insertAlbum.Exec(album.FixNulls())
			if err != nil {
				log.Fatal(err)
			}
			albumId, err = res.LastInsertId()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			albumId = album.Id
		}

		if album == nil {
			log.Fatal("how did this happen?")
		}

		insertAlbumFile.Exec(albumId, fileid)

	}

	tx.Commit()

}

func LookupAlbum(dbx *sqlx.Tx, album string, artist string) (*models.DBAlbum, error) {
	var a models.DBAlbum
	err := dbx.Get(&a, `
    select 
      id,
      album, 
      artist,
      totaltracks,
      discnumber,
      totaldiscs,
      year,
      toyear
    from albums
    where album = ? and artist = ?
  `, album, artist)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return &a, nil
	}
}
