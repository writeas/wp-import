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

var (
	// Username is the username of the account to import to.
	Username string
	// DstBlog is the alias of the blog to import to.
	DstBlog string
	// InstanceURL is the fully qualified URL of the WriteFreely instance to import to.
	InstanceURL string
)
