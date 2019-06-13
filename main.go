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
	"bufio"
	"bytes"
	"fmt"
	"github.com/frankbille/go-wxr-import"
	"github.com/writeas/godown"
	"github.com/writeas/nerds/store"
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

// Print the usage spec to the terminal and exit cleanly
func printUsage(help bool) {
	usage := "usage: wp-import [-h|--help] [-i instance] [-f] filename.xml"
	if help {
		usage = usage + "\n" +
			"  -h|--help     Prints this help message.\n" +
			"  -i            Specifies the instance to use.\n" +
			"                Should be one of the instances set up in instances.ini.\n" +
			"                Defaults to \"writeas\" (https://write.as).\n" +
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

type instance struct {
	Name  string
	Url   string
	Token string
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
	a := parseArgs(os.Args)
	// if len(os.Args) < 2 {
	// 	//errQuit("usage: wp-import https://write.as filename.xml")
	// 	errQuit("usage: wp-import filename.xml")
	// }
	// fname := os.Args[1]
	fname := a["filename"]
	inst := "writeas"
	if a["instance"] != "" {
		inst = a["instance"]
	}

	instances := importConfig()
	//fmt.Println(instances)
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
		fmt.Print("Instance URL: ")
		r.Scan()
		url := r.Text()
		if string(url[:5]) != "https" {
			url = "https://" + url
		}
		if string(url[len(url)-1:]) == "/" {
			url = string(url[:len(url)-1])
		}
		//fmt.Println("Using URL", url)
		fmt.Print("Username: ")
		r.Scan()
		uname := r.Text()
		//fmt.Println("Using username", uname)
		fmt.Print("Password: ")
		r.Scan()
		passwd := r.Text()
		//fmt.Println("Using password", passwd)
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
		fmt.Println("Okay, you're logged in.")
	}

	log.Printf("Reading %s...\n", fname)
	raw, _ := ioutil.ReadFile(fname)

	log.Println("Parsing...")
	d := wxr.ParseWxr(raw)
	log.Printf("Found %d channels.\n", len(d.Channels))

	postsCount := 0

	for _, ch := range d.Channels {
		ch.Title = ch.Title + " " + store.GenerateFriendlyRandomString(4)
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
