/*
 * wp-import imports posts from WordPress into Write.as / WriteFreely.
 * Copyright © 2019 A Bunch Tell LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package main

import (
	"bytes"
	"fmt"
	"github.com/frankbille/go-wxr-import"
	"github.com/writeas/godown"
	"go.code.as/writeas.v2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

var (
	commentReg = regexp.MustCompile("(?m)<!-- ([^m ]|m[^o ]|mo[^r ]|mor[^e ])+ -->\n?")
)

// Print the usage spec to the terminal and exit cleanly
func printUsage(help bool) {
	usage := "usage: wp-import [-h|--help] [-i instance] [-f] filename.xml"
	if help {
		usage = usage + "\n" +
			"  -h|--help     Prints this help message.\n" +
			"  -i            Specifies the instance to use.\n" +
			"                Should include the protocol prefix (e.g. https://).\n" +
			"                Defaults to https://write.as .\n" +
			"  -f            Specifies the filename to read from.\n" +
			"                This can be a relative or absolute path.\n" +
			"                The flag can be excluded if the filename is the last argument."
	}
	fmt.Println(usage)
	os.Exit(0)
}

// This should allow input in these formats:
//   wp-import -h (or --help)
//   wp-import filename
//   wp-import -i instance filename
//   wp-import -i instance -f filename

func parseArgs(args []string) map[string]string {
	arguments := make(map[string]string)
	if len(args) == 2 {
		if args[1] == "-h" || args[1] == "--help" {
			printUsage(true)
		} else if string(args[1][0]) != "-" {
			arguments["filename"] = args[1]
		} else {
			printUsage(false)
		}
	} else if len(args) < 2 {
		printUsage(false)
	} else {
		// Starting at 1 because args[0] is the program name
		for i := 1; i < len(args); i++ {
			if args[i] == "-h" || args[i] == "--help" {
				printUsage(true)
			} else if args[i] == "-i" {
				if i+1 == len(args) || string(args[i+1][0]) == "-" {
					printUsage(false)
				}
				arguments["instance"] = args[i+1]
				i++
			} else if args[i] == "-f" {
				if i+1 == len(args) || string(args[i+1][0]) == "-" {
					printUsage(false)
				}
				arguments["filename"] = args[i+1]
				i++
			} else if i == len(args)-1 && string(args[i][0]) != "-" {
				arguments["filename"] = args[i]
			}
		}
	}
	if arguments["filename"] == "" {
		printUsage(false)
	}
	return arguments
}

func main() {
	a := parseArgs(os.Args)
	// if len(os.Args) < 2 {
	// 	//errQuit("usage: wp-import https://write.as filename.xml")
	// 	errQuit("usage: wp-import filename.xml")
	// }
	// fname := os.Args[1]
	fname := a["filename"]
	instance := "https://write.as"
	if a["instance"] != "" {
		instance = a["instance"]
	}
	// testing
	fmt.Println(fname, instance)
	os.Exit(0)

	// TODO: load user config from same func as writeas-cli
	t := ""
	if t == "" {
		errQuit("not authenticated. run: writeas auth <username>")
	}

	cl := writeas.NewClientWith(writeas.Config{
		URL:   instance + "/api",
		Token: t,
	})

	log.Printf("Reading %s...\n", fname)
	raw, _ := ioutil.ReadFile(fname)

	log.Println("Parsing...")
	d := wxr.ParseWxr(raw)
	log.Printf("Found %d channels.\n", len(d.Channels))

	postsCount := 0

	for _, ch := range d.Channels {
		log.Printf("Channel: %s\n", ch.Title)

		// Create the blog
		c := &writeas.CollectionParams{
			Title:       ch.Title,
			Description: ch.Description,
		}
		log.Printf("Creating %s...\n", ch.Title)
		coll, err := cl.CreateCollection(c)
		if err != nil {
			errQuit(err.Error())
		}
		log.Printf("Done!\n")

		log.Printf("Found %d items.\n", len(ch.Items))
		//for i, wpp := range ch.Items {
		for _, wpp := range ch.Items {
			if wpp.PostType != "post" {
				continue
			}

			// Convert to Markdown
			b := bytes.NewBufferString("")
			r := bytes.NewReader([]byte(wpp.Content))
			err = godown.Convert(b, r, nil)
			con := b.String()

			// Remove unneeded WordPress comments that take up space, like <!-- wp:paragraph -->
			con = commentReg.ReplaceAllString(con, "")

			// Append tags
			tags := ""
			sep := ""
			for _, cat := range wpp.Categories {
				if cat.Domain != "post_tag" {
					continue
				}
				tags += sep + "#" + cat.DisplayName
				sep = " "
			}
			if tags != "" {
				con += "\n\n" + tags
			}

			p := &writeas.PostParams{
				Title:      wpp.Title,
				Slug:       wpp.PostName,
				Content:    con,
				Created:    &wpp.PostDateGmt,
				Updated:    &wpp.PostDateGmt,
				Font:       "norm",
				Language:   &ch.Language,
				Collection: coll.Alias,
			}
			log.Printf("Creating %s", p.Title)
			_, err = cl.CreatePost(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "create post: %s\n", err)
				continue
			}

			postsCount++
		}
	}
	log.Printf("Created %d posts.\n", postsCount)
}

func errQuit(m string) {
	fmt.Fprintf(os.Stderr, m+"\n")
	os.Exit(1)
}
