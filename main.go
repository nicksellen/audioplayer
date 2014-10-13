package main

import (
	"github.com/codegangsta/cli"
	"mediaplayer/commands"
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
	}
	app.Run(os.Args)

}
