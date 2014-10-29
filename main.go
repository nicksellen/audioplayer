package main

// -tags leveldb icu libstemmer

import (
	"github.com/codegangsta/cli"
	"github.com/nicksellen/audioplayer/commands"
	"github.com/nicksellen/audioplayer/commands/server"
	"github.com/nicksellen/audioplayer/processing"
	"os"
)

func main() {

	app := cli.NewApp()

	app.Name = "mediaplayer"
	app.Usage = "nicks media player"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:  "index",
			Usage: "index a path",
			Action: func(c *cli.Context) {
				commands.Index(c.Args()[0])
			},
		},
		{
			Name:  "import",
			Usage: "import and process a local folder",
			Action: func(c *cli.Context) {
				commands.Import(c.Args()[0])
			},
		},
		{
			Name:  "process",
			Usage: "process",
			Action: func(c *cli.Context) {
				processing.ProcessTracks()
				processing.ProcessAlbums()
			},
		},
		{
			Name:  "index2",
			Usage: "index2 a path",
			Action: func(c *cli.Context) {
				commands.Index2(c.Args()[0])
			},
		},
		{
			Name:  "search",
			Usage: "search",
			Action: func(c *cli.Context) {
				commands.Search(c.Args()[0])
			},
		},
		{
			Name:  "show",
			Usage: "show everything :)",
			Action: func(c *cli.Context) {
				commands.Show()
			},
		},
		{
			Name:  "server",
			Usage: "run a server",
			Action: func(c *cli.Context) {
				commands.Server()
			},
		},
		{
			Name:  "http",
			Usage: "run http server to serve audio files and library json",
			Action: func(c *cli.Context) {
				server.Server2()
			},
		},
	}
	app.Run(os.Args)

}
