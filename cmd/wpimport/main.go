/*
 * wp-import imports posts from WordPress into Write.as / WriteFreely.
 * Copyright © 2019, 2022 Musing Studio LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package main

import (
	"fmt"
	"github.com/writeas/wp-import/core"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

var fname string

func main() {
	app := &cli.App{
		Name:     "WordPress-to-WriteFreely importer",
		HelpName: "wp-import",
		Usage:    "Import a WordPress blog into Write.as/WriteFreely by running this importer on an exported WXR file.",
		Version:  "1.0.0",
		Commands: []*cli.Command{
			{
				Name:   "import",
				Action: CmdImport,
				Usage:  "Import WordPress export file into WriteFreely",
				Flags: append(core.DefaultFlags, []cli.Flag{
					&cli.StringFlag{
						Name:        "filename",
						Aliases:     []string{"f"},
						Usage:       "",
						Required:    true,
						Destination: &fname,
					},
				}...),
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func errQuit(m string) {
	fmt.Fprintf(os.Stderr, m+"\n")
	os.Exit(1)
}
