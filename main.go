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
	"github.com/writeas/go-writeas/v2"
	"github.com/writeas/godown"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

var (
	commentReg = regexp.MustCompile("(?m)<!-- ([^m ]|m[^o ]|mo[^r ]|mor[^e ])+ -->\n?")
)

func main() {
	if len(os.Args) < 2 {
		//errQuit("usage: wp-import https://write.as filename.xml")
		errQuit("usage: wp-import filename.xml")
	}
	//instance := os.Args[1]
	instance := "https://write.as"
	fname := os.Args[1]

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
