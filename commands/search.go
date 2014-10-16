package commands

import (
	//"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"log"
	"os"
)

func Search(q string) {

	dbpath := "db.bleve"

	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		log.Fatal(err)
	}

	index, err := bleve.Open(dbpath)
	if err != nil {
		log.Fatal(err)
	}

	query := bleve.NewMatchQuery(q)
	search := bleve.NewSearchRequest(query)
	searchResults, err := index.Search(search)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("results: %s\n", searchResults)

	//b, _ := json.MarshalIndent(searchResults, "", " ")

	//fmt.Printf("results: %s\n", string(b))

}
