package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/rodrigopv/liveping/internal/ping"
)

func main() {
	app := &cli.App{
		Name:  "liveping",
		Usage: "A real-time ping monitoring tool with WebSocket interface",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen",
				Aliases: []string{"l"},
				Value:   ":8080",
				Usage:   "Address to listen on for web server",
			},
			&cli.DurationFlag{
				Name:    "interval",
				Aliases: []string{"i"},
				Value:   100 * time.Millisecond,
				Usage:   "Ping interval",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Value:   false,
				Usage:   "Enable verbose output for ping responses",
			},
		},
		ArgsUsage: "<target>",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return cli.ShowAppHelp(c)
			}
			
			config := ping.Config{
				TargetHost:   c.Args().First(),
				ListenAddr:   c.String("listen"),
				PingInterval: c.Duration("interval"),
				Verbose:      c.Bool("verbose"),
			}
			return ping.RunServer(config)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
} 