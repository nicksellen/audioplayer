package commands

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	//	"github.com/blevesearch/bleve/search"
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
	kwFields := []string{
		"artist",
		"title",
		"album",
		"albumartist",
		"composer",
		"label",
		"filetype",
	}
	query := bleve.NewMatchQuery(q)
	s := bleve.NewSearchRequest(query)
	s.Fields = []string{"*"}
	for _, kw := range kwFields {
		kwFacet := bleve.NewFacetRequest("kw_"+kw, 10)
		s.AddFacet(kw, kwFacet)
	}
	res, err := index.Search(s)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("results: %s\n", res)

	b, _ := json.MarshalIndent(res, "", " ")

	fmt.Printf("results: %s\n", string(b))

	for _, hit := range res.Hits {
		fmt.Printf("hit: %s\nfields = %s\n\n", hit.ID, hit.Fields)

		for k, v := range hit.Fields {
			fmt.Printf("field: %s => %s\n", k, v)
		}
	}

	for name, facet := range res.Facets {
		fmt.Printf("--------- %s\n", name)
		for _, tf := range facet.Terms {
			fmt.Printf("%s: %d\n", tf.Term, tf.Count)
		}
	}

	fmt.Printf("\n--------------\ntotal: %d\n", res.Total)

}
