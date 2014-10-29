package processing

import (
	"database/sql"
	"fmt"
	"github.com/arbovm/levenshtein"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type PropertyConflictHandler func(string, string) string

func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./db.v2.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func nullString(val string) *sql.NullString {
	return &sql.NullString{val, val != ""}
}

func nullInt64(val string) *sql.NullInt64 {
	if val != "" {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &sql.NullInt64{n, true}
	}
	return &sql.NullInt64{0, false}
}

func enforceNum(m map[string]string, key string) {
	if val, ok := m[key]; ok {
		m[key] = NUM_RE.FindString(val)
	}
}

var YEAR_RE = regexp.MustCompile("[0-9]{4}")
var NUM_RE = regexp.MustCompile("[0-9]+")
var CAP = regexp.MustCompile("[A-Z]")

func propertyMergeWithPropertyConflictHandler(m1 map[string]string, m2 map[string]string, k string, resolver PropertyConflictHandler) {
	a := strings.TrimSpace(m1[k])
	b := strings.TrimSpace(m2[k])
	if a == b {
		// easy
	} else if b == "" {
		// nothing to do m1 already contains a
	} else if b != "" {
		m1[k] = b
	} else {
		m1[k] = resolver(a, b)
	}
}

func propertyMerge(m1 map[string]string, m2 map[string]string, k string) {
	propertyMergeWithPropertyConflictHandler(m1, m2, k, func(a string, b string) string {
		if EqualsCaseInsenstive(a, b) {
			// the only difference is capitalization, use the one with the most capitalization
			if CountOccurances(b, CAP) > CountOccurances(a, CAP) {
				return b
			}
			return a
		} else {
			dist := levenshtein.Distance(a, b)
			if dist < 3 {
				return b
			}
		}
		fmt.Printf("conflict: %s [%s] vs [%s]\n", k, a, b)
		return a
	})
}

func CountOccurances(s string, re *regexp.Regexp) int {
	return len(re.FindAllString(s, -1))
}

func EqualsCaseInsenstive(a string, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}

var puncRe = regexp.MustCompile("[':;/\\[\\]\"*\\-_\\(\\)$&#@^,\\.\\?]")
var multiSpaceRe = regexp.MustCompile("\\ \\ +")

func ToLowerWithoutPunctuation(s string) string {
	s = puncRe.ReplaceAllLiteralString(s, "")
	s = multiSpaceRe.ReplaceAllLiteralString(s, " ")
	s = strings.TrimSpace(strings.ToLower(s))
	return s
}
