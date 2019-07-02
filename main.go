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
	"archive/zip"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/frankbille/go-wxr-import"
	"github.com/howeyc/gopass"
	"github.com/writeas/godown"
	"github.com/writeas/nerds/store"
	"github.com/writeas/web-core/posts"
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
// If the extension is "xml", verify that it's a valid WordPress WXR file:
//   Check to see if the word "WordPress" appears in the first 200 characters.
// If the extension is "zip", parse it as a ZIP file.
func identifyFile(fname string) *ImportedBlogs {
	log.Printf("Reading %s...\n", fname)
	parts := strings.Split(fname, ".")
	extension := strings.ToLower(parts[len(parts)-1])

	if extension == "xml" {
		raw, _ := ioutil.ReadFile(fname)
		rawstr := string(raw[:200])
		if strings.Contains(rawstr, "WordPress") {
			log.Println("This looks like a WordPress file. Parsing...")
			// Changed my mind. Since we're exporting ParseWPFile it should
			// accept the contents of a file and not rely on the importing
			// program to also import wxr.
			// (We can let them import ioutil, it's core.)
			return ParseWPFile(raw)
		} else {
			// It's XML but not WordPress
			errQuit("It's XML, but not in a format I recognize.")
		}
	} else if extension == "zip" {
		log.Println("This looks like a Zip archive. Parsing...")
		return ParseZipFile(fname)
	} else if extension == "json" {
		// TODO: Identify specifically as a WriteFreely JSON file
		log.Println("This looks like a WriteFreely JSON file. Parsing...")
		return ParseWFJSONFile(fname)
	} else {
		errQuit("I can't tell what kind of file this is.")
	}
	// punt
	return &ImportedBlogs{}
}

// Turn our WXR struct into an ImportedBlogs struct.
// Creating a general format for imported blogs is the first step to
// generalizing the import process.
func ParseWPFile(raw []byte) *ImportedBlogs {
	d = wxr.ParseWxr(raw)
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

// Read through the ZIP file, converting text files to posts
// and directories to blogs.
// If the filename ends in a / it's a directory.
// Otherwise, filenames have the format "[directory/]postname.txt"
// If there's no directory, then it's a draft post
// If there is a directory, the directory is the blog the post goes in
func ParseZipFile(fname string) *ImportedBlogs {
	//return &ImportedBlogs{}
	zf, err := zip.OpenReader(fname)
	if err != nil {
		errQuit(err.Error())
	}
	defer zf.Close()

	coll := &ImportedBlogs{}
	t_coll := make(map[string]*SingleBlog{})

	t_coll["Drafts"] = &SingleBlog{
		Params: &writeas.CollectionParams{},
		Posts:  make([]*writeas.PostParams, 0, 0),
	}

	for _, f := range zf.File {
		// A trailing slash means this is an empty directory
		isEmptyDir := strings.HasSuffix(f.Name, "/")
		if isEmptyDir {
			title := f.Name[:len(f.Name)-1]
			// I think this will work. &SingleBlog{} should be the null value
			// If there isn't already a blog with this name, make one
			if (t_coll[title] == &SingleBlog{}) {
				t_coll[title] = &SingleBlog{
					Params: &writeas.CollectionParams{
						Title:       title,
						Description: "",
					},
					Posts: make([]*writeas.PostParams, 0, 0),
				}
			}
			// If there is, we don't need to do anything.
			// Either way, skip the rest of the block and go to the next file
			continue
		}

		// Get directory, slug, etc. from the filename
		fParts := strings.Split(f.Name, "/")
		var collAlias string
		var postFname string
		if len(fParts) == 1 {
			// This is a top-level file
			collAlias = "Drafts"
			postFname = fParts[0]
		} else {
			// This is a collection post
			collAlias = fParts[0]
			postFname = fParts[1]
		}

		// Ideally, we'll reach each collection's directory before we reach
		// the first post in the collection. But we can't rely on a zip
		// file's ordering to be deterministic. So just in case, we do this
		// check twice.
		if (t_coll[collAlias] == &SingleBlog{}) {
			t_coll[collAlias] = &SingleBlog{
				Params: &writeas.CollectionParams{
					Title:       collAlias,
					Description: "",
				},
				Posts: make([]*writeas.PostParams, 0, 0),
			}
		}

		// Get file contents
		fc, err := f.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't open file: %s: %v", f.Name, err)
			continue
		}
		defer fc.Close()
		content, err := ioutil.ReadAll(fc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Opened file but couldn't read it: %s: %v", f.Name, err)
			continue
		}

		// Build post parameters
		p := filenameToParams(postFname)
		p.Created = &f.Modified
		p.Title, p.Content = posts.ExtractTitle(string(content))

		t_coll[collAlias].Posts = append(t_coll[collAlias].Posts, p)

		fmt.Printf("%s - %s - %+v\n", f.Name, collAlias, p)
	}

	for k, v := range t_coll {
		coll.Collections = append(coll.Collections, v)
	}
	return coll
}

// Turn WriteFreely JSON file into an ImportedBlogs struct.
// TODO: Find out how our JSON files are structured!
//
func ParseWFJSONFile(fname string) *ImportedBlogs {
	return &ImportedBlogs{}

	return nil
}

// filenameToParams returns PostParams with the ID and slug derived from the given filename.
func filenameToParams(fname string) *writeas.PostParams {
	baseParts := strings.Split(fname, ".")
	// This assumes there's at least one '.' in the filename, e.g. abc123.txt
	// TODO: handle the case where len(baseParts) != 2
	baseName := baseParts[0]

	p := &writeas.PostParams{}

	parts := strings.Split(baseName, "_")
	if len(parts) == 1 {
		// There's no slug -- only an ID
		p.ID = parts[0]
	} else {
		// len(parts) > 1
		p.Slug = parts[0]
		p.ID = parts[1]
	}
	return p
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
