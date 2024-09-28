/*
 * wp-import imports posts from WordPress into Write.as / WriteFreely.
 * Copyright © 2024 Musing Studio LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package core

import (
	"fmt"
	"log"
	"net/url"

	"github.com/writeas/go-writeas/v2"
)

var Client *writeas.Client

func SignIn(u, p, i string) error {
	if i == "" {
		fmt.Println("Logging in...")
		// Client = writeas.NewDevClient()		// Use in development
		Client = writeas.NewClient()
		_, err := Client.LogIn(u, p)
		if err != nil {
			return err
		}
		fmt.Println("Logged in!")
		return nil
	}

	instance, err := url.Parse(i)
	if err != nil {
		return err
	}
	instance.Scheme = "https"
	instance.Path += "/api"

	fmt.Println("Logging in to", i)
	config := writeas.Config{URL: instance.String()}
	Client = writeas.NewClientWith(config)
	_, err = Client.LogIn(u, p)
	if err != nil {
		return err
	}
	fmt.Println("Logged in!")
	return nil
}

func SignOut() {
	fmt.Println("Logging out...")
	err := Client.LogOut()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logged out!")
}
