package commands

import (
	"code.google.com/p/go-uuid/uuid"
	"container/list"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketConnection struct {
	Connection *websocket.Conn
	Id         string
}

func Server() {

	db, err := sql.Open("sqlite3", "./db.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := mux.NewRouter()

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/index.html")
	}

	// all the pages react can handle, just give it the index.html
	router.HandleFunc("/", serveIndex)
	router.HandleFunc("/albums", serveIndex)
	router.HandleFunc("/albums/{name:.*}", serveIndex)
	router.HandleFunc("/player", serveIndex)

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
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			var name string
			rows.Scan(&name)
			artists = append(artists, Artist{Name: name})
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
		name := vars["name"]
		fmt.Printf("looking album: %s\n", name)
		rows, err := db.Query(`
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
		w.Header().Set("Content-Type", FormatToContentType(path[len(path)-3:]))
		http.ServeFile(w, r, store+path)
	})

	http.Handle("/", router)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	wsconns := list.New()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		wsconn := WebsocketConnection{Connection: conn, Id: uuid.New()}
		wsconns.PushBack(wsconn)
		fmt.Printf("%d ws connections\n", wsconns.Len())

		m := make(map[string]interface{})
		m["type"] = "connected"
		m["id"] = wsconn.Id
		WebsocketBroadcast(wsconns, wsconn, 1, m)

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("error reading message: %s\n", err)
				for e := wsconns.Front(); e != nil; e = e.Next() {
					c := e.Value.(WebsocketConnection)
					if c == wsconn {
						wsconns.Remove(e)
						m := make(map[string]interface{})
						m["type"] = "disconnected"
						m["id"] = wsconn.Id
						WebsocketBroadcast(wsconns, wsconn, 1, m)
					}
				}
				return
			}
			var m map[string]interface{}
			json.Unmarshal(p, &m)
			m["id"] = wsconn.Id
			WebsocketBroadcast(wsconns, wsconn, messageType, m)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func WebsocketBroadcast(conns *list.List, except WebsocketConnection, messageType int, m map[string]interface{}) {
	b, err := json.Marshal(m)
	if err != nil {
		log.Printf("error writing json: %s\n", err)
		return
	}
	for e := conns.Front(); e != nil; e = e.Next() {
		c := e.Value.(WebsocketConnection)
		if c != except {
			if err := c.Connection.WriteMessage(messageType, b); err != nil {
				log.Println(err)
			}
		}
	}
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
