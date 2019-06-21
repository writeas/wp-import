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
	//	"archive/zip"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/frankbille/go-wxr-import"
	"github.com/howeyc/gopass"
	"github.com/writeas/godown"
	"github.com/writeas/nerds/store"
	//	"github.com/writeas/web-core/posts"
	//	"github.com/writeas/wf-migrate"
	// "github.com/writeas/zip-import"
	"go.code.as/writeas.v2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	commentReg = regexp.MustCompile("(?m)<!-- ([^m ]|m[^o ]|mo[^r ]|mor[^e ])+ -->\n?")
)

type instance struct {
	Name  string
	Url   string
	Token string
}

type ImportedBlogs struct {
	Collections []*SingleBlog
}

type SingleBlog struct {
	Params *writeas.CollectionParams
	Posts  []*writeas.PostParams
}

// Preparing to be able to handle multiple types of files.
// For right now, just verify that it's a valid WordPress WXR file.
// Do this two ways: check that the file extension is "xml", and
// verify that the word "WordPress" appears in the first 200 characters.
func identifyFile(fname string) *ImportedBlogs {
	log.Printf("Reading %s...\n", fname)
	parts := strings.Split(fname, ".")
	extension := parts[len(parts)-1]

	if extension == "xml" {
		raw, _ := ioutil.ReadFile(fname)
		rawstr := string(raw[:200])
		if strings.Contains(rawstr, "WordPress") {
			log.Println("This looks like a WordPress file. Parsing...")
			// Since I know it's a WP file I might as well do this here
			// instead of delegating wxr.ParseWxr to the helper function
			return ParseWPFile(wxr.ParseWxr(raw))
		} else {
			// It's XML but not WordPress
			errQuit("It's XML, but not in a format I recognize.")
		}
		// Future development:
		//} else if extension == "zip" {
		//	log.Println("This looks like a Zip archive. Parsing...")
		//	return ParseZipFile(fname)
	} else {
		errQuit("I can't tell what kind of file this is.")
	}
	// punt
	return &ImportedBlogs{}
}

// Turn our WXR struct into an ImportedBlogs struct.
// Creating a general format for imported blogs is the first step to
// generalizing the import process.
func ParseWPFile(d wxr.Wxr) *ImportedBlogs {
	coll := &ImportedBlogs{}
	for _, ch := range d.Channels {
		// Create the blog
		c := &SingleBlog{
			Params: &writeas.CollectionParams{
				Title:       ch.Title,
				Description: ch.Description,
			},
			Posts: make([]*writeas.PostParams, 0, 0),
		}

		for _, wpp := range ch.Items {
			if wpp.PostType != "post" {
				continue
			}

			// Convert to Markdown
			b := bytes.NewBufferString("")
			r := bytes.NewReader([]byte(wpp.Content))
			err := godown.Convert(b, r, nil)
			if err != nil {
				errQuit(err.Error())
			}
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
			var postlang string
			if len(ch.Language) > 2 {
				postlang = string(ch.Language[:2])
			} else {
				postlang = ch.Language
			}
			p := &writeas.PostParams{
				Title:    wpp.Title,
				Slug:     wpp.PostName,
				Content:  con,
				Created:  &wpp.PostDateGmt,
				Updated:  &wpp.PostDateGmt,
				Font:     "norm",
				Language: &postlang,
			}
			c.Posts = append(c.Posts, p)
		}
		coll.Collections = append(coll.Collections, c)
	}
	return coll
}

// Temporarily using an ini file to store instance tokens.
// This is probably not what the rest of the code does,
// but I need some way to handle this for now.
// TODO: Get this in line with the rest of the code (see T586)

// ini file format:
// Each instance has its own [section]
// Semicolons (;) at the beginning of a line indicate a comment
// Can't start a comment mid-line (this allows semicolons in variable values)
// Blank lines are ignored
func importConfig() map[string]instance {
	file, err := ioutil.ReadFile("instances.ini")
	if err != nil {
		errQuit("Error reading instances.ini")
	}
	lines := strings.Split(string(file), "\n")
	instances := make(map[string]instance)
	curinst := ""
	newinst := instance{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		fc := string(line[0])
		if fc == ";" {
			continue
		}
		if fc == "[" {
			if curinst != "" {
				instances[curinst] = newinst
				newinst = instance{}
			}
			curinst = line[1:(len(line) - 1)]
		} else {
			loc := strings.Index(line, "=")
			if curinst == "" || loc == -1 {
				errQuit("Malformed ini file")
			}
			k := line[:loc]
			v := line[loc+1:]
			if k == "url" {
				newinst.Url = v
			} else if k == "token" {
				newinst.Token = v
			} else {
				errQuit("Malformed ini file")
			}
		}
	}
	instances[curinst] = newinst
	return instances
}

func main() {
	f_inst := flag.String("i", "writeas", "Named WriteFreely Host (not URL)")
	f_file := flag.String("f", "", "File to be imported")
	f_help := flag.Bool("h", false, "Print this help message")
	f_dry := flag.Bool("d", false, "Dry run (parse the input file but don't upload the contents)")
	f_verb := flag.Bool("v", false, "Display all messages instead of just important ones")
	flag.Parse()
	a := flag.Args()
	if (*f_file == "" && len(a) == 0) || (*f_help == true) {
		fmt.Fprintf(os.Stderr, "usage: wfimport [-i myinstance] [-f] file1\n")
		flag.PrintDefaults()
		return
	}
	vbs := *f_verb
	var fname string
	if *f_file != "" {
		fname = *f_file
	} else {
		fname = a[0]
	}
	inst := "writeas"
	if *f_inst != "" {
		inst = *f_inst
	}

	instances := importConfig()
	var cl *writeas.Client
	t := ""
	u := ""
	if val, ok := instances[inst]; ok {
		t = val.Token
		u = val.Url
		cl = writeas.NewClientWith(writeas.Config{
			URL:   u + "/api",
			Token: t,
		})
	} else {
		fmt.Println("We don't have a token for " + inst + ".")
		r := bufio.NewScanner(os.Stdin)
		fmt.Print("Instance URL (include http/https): ")
		r.Scan()
		url := r.Text()
		if string(url[:4]) != "http" {
			url = "https://" + url
		}
		if string(url[len(url)-1:]) == "/" {
			url = string(url[:len(url)-1])
		}
		fmt.Print("Username: ")
		r.Scan()
		uname := r.Text()
		fmt.Print("Password: ")
		tpwd, pwerr := gopass.GetPasswdMasked()
		if pwerr != nil {
			errQuit(pwerr.Error())
		}
		passwd := string(tpwd)
		cl = writeas.NewClientWith(writeas.Config{
			URL:   url + "/api",
			Token: "",
		})
		_, uerr := cl.LogIn(uname, passwd)
		if uerr != nil {
			errQuit("Couldn't log in with those credentials.")
		}
		file, _ := os.OpenFile("instances.ini", os.O_APPEND|os.O_WRONLY, 0644)
		defer file.Close()
		printstr := "\n[" + inst + "]\nurl=" + url + "\ntoken=" + cl.Token()
		fmt.Fprintln(file, printstr)
		if vbs {
			fmt.Println("Okay, you're logged in.")
		}
	}

	// What kind of file is it?
	d := identifyFile(fname) // d is now an ImportedBlogs object, not a WXR object

	log.Printf("Found %d channels.\n", len(d.Collections))

	postsCount := 0

	for _, ch := range d.Collections {
		c := ch.Params
		title := c.Title
		log.Printf("Channel: %s\n", title)
		var coll *writeas.Collection
		var err error
		if *f_dry == false {
			log.Printf("Creating %s...\n", title)
			coll, err = cl.CreateCollection(c)
			if err != nil {
				if err.Error() == "Collection name is already taken." {
					title = title + " " + store.GenerateFriendlyRandomString(4)
					log.Printf("A blog by that name already exists. Changing to %s...\n", title)
					c.Title = title
					coll, err = cl.CreateCollection(c)
					if err != nil {
						errQuit(err.Error())
					}
				} else {
					errQuit(err.Error())
				}
			}
			if vbs {
				log.Printf("Done!\n")
			}
		}
		log.Printf("Found %d posts.\n", len(ch.Posts))
		for _, p := range ch.Posts {
			if vbs {
				log.Printf("Creating %s", p.Title)
			}
			if *f_dry == false {
				p.Collection = coll.Alias
				_, err = cl.CreatePost(p)
				if err != nil {
					fmt.Fprintf(os.Stderr, "create post: %s\n", err)
					continue
				}
			}
			postsCount++
		}
	}
	log.Printf("Created %d posts.\n", postsCount)
	if *f_dry == true {
		log.Println("THIS WAS A DRY RUN! No posts or collections were actually created on the remote server.")
	}
}

func errQuit(m string) {
	fmt.Fprintf(os.Stderr, m+"\n")
	os.Exit(1)
}
