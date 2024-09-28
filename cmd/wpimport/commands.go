package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	wpimport "github.com/writeas/wp-import"
	"github.com/writeas/wp-import/core"
	"golang.org/x/term"
	"io/ioutil"
	"log"
)

func CmdImport(c *cli.Context) error {
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
}
