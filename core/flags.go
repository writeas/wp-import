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

import "github.com/urfave/cli/v2"

var DefaultFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "user",
		Aliases:     []string{"u"},
		Usage:       "The username for the Write.as/WriteFreely account",
		Required:    true,
		Destination: &Username,
	},

	&cli.StringFlag{
		Name:        "blog",
		Aliases:     []string{"b"},
		Usage:       "The alias of the destination blog for importing your content.",
		Required:    true,
		Destination: &DstBlog,
	},

	&cli.StringFlag{
		Name:        "instance",
		Aliases:     []string{"i"},
		Usage:       "Provide the URL of your WriteFreely instance (e.g., '--instance https://pencil.writefree.ly')",
		Destination: &InstanceURL,
	},
}
