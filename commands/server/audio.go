package server

import (
	"database/sql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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
