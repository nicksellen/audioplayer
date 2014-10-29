package processing

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/kr/pretty"
	"github.com/nicksellen/audioplayer/models"
	"log"
	"strings"
)

type DBImportAlbum struct {
	*models.DBAlbum
	TrackIds string
}

func ProcessAlbums() {

	db := OpenDB()
	dbx := sqlx.NewDb(db, "sqlite3")
	defer db.Close()

	dbx.MustExec(`

    drop table if exists albums;

    create table if not exists albums (

      id integer primary key,

      album           text,
      albumartist     text,
      artists         text,

      compilation     text,

      year            int,
      toyear          int,

      totaltracks     int,

      discnumber      int,
      totaldiscs      int

    );
  `)
	albums := []DBImportAlbum{}

	err := dbx.Select(&albums, `
  
    -- woah looks hairy eh? take a deep breath...

    select
      coalesce(albumartist, 'Various Artists') as albumartist,
      --nullif(artists, albumartist) as artists,
      artists as artists,
      album,

      group_concat(ids) as trackids,

      min(totaltracks)  as totaltracks,
      min(discnumber)   as discnumber,
      min(totaldiscs)   as totaldiscs,
      min(year)         as year,
      max(toyear)       as toyear

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

        group_concat(distinct t.id) as ids,

        -- include a mention all the other artists
        group_concat(distinct t.artist) as artists,

        t.album, 

        min(t.discnumber) as discnumber,
        min(t.totaldiscs) as totaldiscs,

        min(t.totaltracks) as totaltracks,

        min(year) year,
        nullif(max(year), min(year)) toyear

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

	tx := dbx.MustBegin()

	insertAlbum, err := tx.PrepareNamed(`
    insert into albums (
      album, albumartist, artists, year, totaltracks, discnumber, totaldiscs
    ) values (
      :album, :albumartist, :artists, :year, :totaltracks, :discnumber, :totaldiscs
    )
  `)
	if err != nil {
		log.Fatal(err)
	}

	m := make(map[string]DBImportAlbum)

	for _, album := range albums {
		a := album.ToAlbum()
		s := ToLowerWithoutPunctuation(a.Album)
		if e, exists := m[s]; exists {

			trackIds := album.TrackIds

			b := e.ToAlbum()

			/*
			   - does `albumartist` of one contain the `albumartist` of the other?
			   - does `artists` of one equal the `albumartist` of the other?
			   - does `artists` of one contain the `albumartist` of the other?
			*/

			fmt.Printf("already got: %s\n", a.Album)
			for _, diff := range pretty.Diff(a, b) {
				fmt.Printf("  %s\n", diff)
			}

			merge := false

			if a.AlbumArtist != "" && a.AlbumArtist == b.AlbumArtist {
				merge = true
			} else if a.Artists != "" && a.Artists == b.Artists {
				merge = true
			} else if a.Artists != "" && a.Artists == b.AlbumArtist {
				a.AlbumArtist = a.Artists
				a.Artists = ""

				merge = true
			} else if b.Artists != "" && b.Artists == a.AlbumArtist {
				merge = true

			} else if OneContainsTheOther(a.AlbumArtist, b.AlbumArtist) {
				a.AlbumArtist = ShortestOf(a.AlbumArtist, b.AlbumArtist)
				a.Artists = a.Artists + "," + b.Artists
				merge = true
			} else if OneContainsTheOther(a.AlbumArtist, b.Artists) {
				a.AlbumArtist = b.AlbumArtist
				a.Artists = a.Artists + "," + b.Artists
				merge = true
			} else if OneContainsTheOther(b.AlbumArtist, a.Artists) {
				a.Artists = a.Artists + "," + b.Artists
				merge = true
			}

			if merge {
				a.Album = ShortestOf(a.Album, b.Album)
				trackIds = trackIds + "," + e.TrackIds
				a.TotalTracks = MinOf(a.TotalTracks, b.TotalTracks)
				a.DiscNumber = MinOf(a.DiscNumber, b.DiscNumber)
				a.TotalDiscs = MinOf(a.TotalDiscs, b.TotalDiscs)
				a.Year = MinOf(a.Year, b.Year)
				a.ToYear = MinOf(a.ToYear, b.ToYear)
				fmt.Printf(" trackIds: %v\n", trackIds)
				fmt.Printf("\n final: %#  v\n", pretty.Formatter(a))
			}
			fmt.Printf(" merge: %v\n", merge)
			fmt.Printf("\n\n")

		} else {
			m[s] = album
		}

	}

	for _, album := range m {
		_, err = insertAlbum.Exec(album.FixNulls())
		if err != nil {
			log.Fatal(err)
		}
	}

	tx.Commit()

}

func OneContainsTheOther(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

func ShortestOf(a, b string) string {
	if len(a) < len(b) && a != "" {
		return a
	}
	return b
}

func MinOf(a, b int64) int64 {
	if a < b && a != 0 {
		return a
	}
	return b
}
