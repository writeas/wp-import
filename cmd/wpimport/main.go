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
	"io/ioutil"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	wpimport "github.com/writeas/wp-import"
	"github.com/writeas/wp-import/core"
	"golang.org/x/term"
)

func main() {
	var fname string

	app := &cli.App{
		Name:    "WriteFreely WordPress importer",
		Usage:   "Import a WordPress blog into Write.as/WriteFreely by running this importer on an exported WXR file.",
		Version: "1.0.0",
		Flags: append(core.DefaultFlags, []cli.Flag{
			&cli.StringFlag{
				Name:        "filename",
				Aliases:     []string{"f"},
				Usage:       "",
				Required:    true,
				Destination: &fname,
			},
		}...),
		Action: func(c *cli.Context) error {
			fmt.Println("Hello", core.Username)

			fmt.Println("Please enter your password:")
			var enteredPassword string
			for {
				password, err := term.ReadPassword(0)
				if err != nil {
					panic(err)
				}
				if len(password) != 0 {
					fmt.Println("Press Return to log in and start the migration.")
					enteredPassword = string(password)
				} else {
					break
				}
			}
			err := core.SignIn(core.Username, enteredPassword, core.InstanceURL)
			if err != nil {
				log.Fatal(err)
			}

			//fmt.Println("Importing content from content ->", srcPath)
			fmt.Println("Importing content into blog alias ->", core.DstBlog)

			log.Printf("Reading %s...\n", fname)
			raw, err := ioutil.ReadFile(fname)
			if err != nil {
				core.SignOut()
				log.Fatal(err)
			}

			log.Println("Parsing...")
			err = wpimport.ImportWordPress(core.DstBlog, raw)
			if err != nil {
				core.SignOut()
				log.Fatal(err)
			}
			/*
				posts, err := ParseContentDirectory(srcPath, uploadImages)
				if err != nil {
					SignOut()
					log.Fatal(err)
				}
			*/

			/*
				fmt.Println("")
				fmt.Println("Publishing posts to", core.DstBlog)
				for _, post := range posts {
					err := PublishPost(post, core.DstBlog)
					if err != nil {
						core.SignOut()
						log.Fatal(err)
					}
				}
				fmt.Println("Posts published.")
			*/

			core.SignOut()

			/*
				err = WriteResponsesToDisk()
				if err != nil {
					log.Fatal(err)
				}
			*/

			return nil
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
