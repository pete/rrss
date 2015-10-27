// RSS feed reader that outputs plain text, werc/apps/barf, or werc/apps/blagh format.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	// BUG: New versions of SlyMarbo/rss intermittently
	// return an empty Item.Link value. Now using a version
	// dowloaded on 2014.02.04. TODO: Vendor it? Whatever?
	"github.com/SlyMarbo/rss"
)

var (
	debug  = flag.Bool("d", false, "print debug msgs to stderr")
	format = flag.String("f", "", "output format")
	root   = flag.String("r", "", "output root")
	tag    = flag.String("t", "", "feed tag (barf only)")
	url    = flag.String("u", "", "feed url")
)

func usage() {
	os.Stderr.WriteString("usage: rrss [-f barf|blagh] [-r root] [-t tag] [-u url]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func isold(link string, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(link, scanner.Text()) {
			return true
		}
	}
	return false
}

func makeold(link string, path string) (int, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0775)
	defer f.Close()
	check(err)
	if link == "" {
		link = "empty"
	}
	return f.WriteString(link + "\n")
}

// https://bitbucket.org/stanleylieber/barf
func barf(url string) {
	feed, err := rss.Fetch(url)
	check(err)
	for _, i := range feed.Items {
		d := "src"
		links := "links"
		if *root != "" {
			d = *root + "/" + d
			links = *root + "/" + links
		}
		if isold(i.Link, links) {
			continue
		}
		err = os.MkdirAll(d, 0775)
		check(err)
		f, err := os.Open(d)
		defer f.Close()
		check(err)
		dn, err := f.Readdirnames(0)
		check(err)
		var di []int
		for _, j := range dn {
			k, _ := strconv.Atoi(j)
			di = append(di, k)
		}
		sort.Ints(di)
		n := 1
		if di != nil {
			n = di[len(di)-1] + 1
		}
		d = fmt.Sprintf("%s/%d", d, n)
		if *debug == true {
			fmt.Printf("%s len(di): %d n: %d d: %s\n",
				i.Link, len(di), n, d)
		}
		err = os.MkdirAll(d, 0775)
		check(err)
		err = ioutil.WriteFile(d+"/title", []byte(i.Title+"\n"), 0775)
		check(err)
		err = ioutil.WriteFile(d+"/link", []byte(i.Link+"\n"), 0775)
		check(err)
		err = ioutil.WriteFile(d+"/date", []byte(i.Date.String()+"\n"), 0775)
		check(err)
		err = ioutil.WriteFile(d+"/body", []byte(i.Content+"\n"), 0775)
		check(err)
		if *tag != "" {
			err = os.MkdirAll(d+"/tags", 0775)
			check(err)
			for _, j := range strings.Split(*tag, " ") {
				f, err := os.Create(d + "/tags/" + j)
				f.Close()
				check(err)
			}
		}
		_, err = makeold(i.Link, links)
		check(err)
	}
}

// http://werc.cat-v.org/apps/blagh
func blagh(url string) {
	feed, err := rss.Fetch(url)
	check(err)
	for _, i := range feed.Items {
		d := fmt.Sprintf("%d/%02d/%02d", i.Date.Year(), i.Date.Month(), i.Date.Day())
		links := "links"
		if *root != "" {
			d = *root + "/" + d
			links = *root + "/" + links
		}
		if isold(i.Link, links) {
			continue
		}
		f, _ := os.Open(d) // directory will usually not exist yet
		defer f.Close()
		n, _ := f.Readdirnames(0)
		d = fmt.Sprintf("%s/%d", d, len(n))
		err = os.MkdirAll(d, 0775)
		check(err)
		err = ioutil.WriteFile(
			d+"/index.md",
			[]byte(i.Title+"\n========\n\n"+i.Content+"\n"),
			0775,
		)
		check(err)
		_, err = makeold(i.Link, links)
		check(err)
	}
}

func stdout(url string) {
	feed, err := rss.Fetch(url)
	check(err)
	for _, i := range feed.Items {
		fmt.Printf("title: %s\nlink: %s\ndate: %s\n%s\n\n",
			i.Title, i.Link, i.Date, i.Content)
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *url == "" {
		usage()
	}
	switch *format {
	case "barf":
		barf(*url)
	case "blagh":
		blagh(*url)
	case "":
		stdout(*url)
	default:
		usage()
	}
}
