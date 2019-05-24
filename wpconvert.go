package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	xj "github.com/basgys/goxml2json"
)

type AboutPkg struct {
	Name    string
	Version string
	Author  string
	Owner   string
	Repo    string
}

func ConvertXMLToJson(intxt []uint8) string {
	xml := strings.NewReader(string(intxt))
	json, converr := xj.Convert(xml)
	if converr != nil {
		panic(converr)
	}
	return json.String()
}

func ConvertXMLFileToJson(infn string) string {
	dat, readerr := ioutil.ReadFile(infn)
	if readerr != nil {
		panic(readerr)
	}
	return ConvertXMLToJson(dat)
}

func ConvertXMLFileToJsonFile(infn, outfn string) {
	json := []byte(ConvertXMLFileToJson(infn))
	writeerr := ioutil.WriteFile(outfn, json, 0666)
	if writeerr != nil {
		panic(writeerr)
	}
	fmt.Println("Imported " + infn)
	fmt.Println("Exported " + outfn)
}

func main() {
	abt := AboutPkg{"WordPress Converter", "0.1.1", "Noëlle Anthony", "WriteAs / A Bunch Tell", "https://github.com/writeas/wp-import"}
	if len(os.Args) != 3 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			fmt.Println(abt.Name, abt.Version)
			fmt.Println("Repository: ", abt.Repo)
			fmt.Println("Written by: ", abt.Author)
			fmt.Println("Publisher : ", abt.Owner)
		} else {
			fmt.Println(abt.Name, abt.Version)
		}
		fmt.Println("Usage: ./wpconvert /path/to/infile /path/to/outfile")
	} else {
		ConvertXMLFileToJsonFile(os.Args[1], os.Args[2])
	}
}
