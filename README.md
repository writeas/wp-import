# Import WordPress blog to Write.as

_**This utility is a work in progress.**_

This tool reads a WordPress Extended RSS (WXR) file, creates a new Write.as / WriteFreely blog with the same information, and imports all posts.

## Usage

```
NAME:
   wp-import - Import a WordPress blog into Write.as/WriteFreely by running this importer on an exported WXR file.

USAGE:
   wp-import [global options] command [command options]

VERSION:
   1.0.0

COMMANDS:
   import   Import WordPress export file into WriteFreely
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --user value, -u value      Username for the Write.as/WriteFreely account
   --blog value, -b value      Alias of the destination blog for importing your content.
   --instance value, -i value  URL of your WriteFreely instance (e.g., '--instance https://pencil.writefree.ly') (default: https://write.as)
   --filename value, -f value  
   --help, -h                  show help
   --version, -v               print the version
```