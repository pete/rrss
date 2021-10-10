// RSS feed reader that outputs plain text, werc/apps/barf, or werc/apps/blagh format.
package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/SlyMarbo/rss"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
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

func fetchfeed(url string) (resp *http.Response, err error) {
	defaultTransport := http.DefaultTransport.(*http.Transport)

	// Create new Transport that ignores self-signed SSL
	customTransport := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		DialContext:           defaultTransport.DialContext,
		MaxIdleConns:          defaultTransport.MaxIdleConns,
		IdleConnTimeout:       defaultTransport.IdleConnTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: customTransport}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; hjdicks)")
	return client.Do(req)
}

func isold(date time.Time, link string, path string) bool {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0775)
	if err != nil {
		return true
	}
	defer file.Close()
	s := fmt.Sprintf("%d_%s", date.Unix(), link)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(s, scanner.Text()) {
			return true
		}
	}
	return false
}

func makeold(date time.Time, link string, path string) (int, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0775)
	defer f.Close()
	check(err)
	if link == "" {
		link = "empty"
	}
	s := fmt.Sprintf("%d_%s", date.Unix(), link)
	return f.WriteString(s + "\n")
}

// https://code.9front.org/hg/barf
func barf(url string) {
	feed, err := rss.FetchByFunc(fetchfeed, url)
	if *debug {
		log.Printf("Tried fetching feed '%s' => err: %v\n", url, err)
	}
	check(err)
	for _, i := range feed.Items {
		d := "src"
		links := "links"
		if *root != "" {
			d = *root + "/" + d
			links = *root + "/" + links
		}
		if isold(i.Date, i.Link, links) {
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
		err = ioutil.WriteFile(d+"/body", []byte(conorsum(i)+"\n"), 0775)
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
		_, err = makeold(i.Date, i.Link, links)
		check(err)
	}
}

// http://werc.cat-v.org/apps/blagh
func blagh(url string) {
	feed, err := rss.FetchByFunc(fetchfeed, url)
	check(err)
	for _, i := range feed.Items {
		d := fmt.Sprintf("%d/%02d/%02d", i.Date.Year(), i.Date.Month(), i.Date.Day())
		links := "links"
		if *root != "" {
			d = *root + "/" + d
			links = *root + "/" + links
		}
		if isold(i.Date, i.Link, links) {
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
			[]byte(i.Title+"\n===\n\n"+conorsum(i)+"\n"),
			0775,
		)
		check(err)
		_, err = makeold(i.Date, i.Link, links)
		check(err)
	}
}

func stdout(url string) {
	feed, err := rss.FetchByFunc(fetchfeed, url)
	if *debug {
		log.Printf("Tried fetching feed '%s' => err: %v\n", url, err)
	}
	check(err)
	for _, i := range feed.Items {
		fmt.Printf("title: %s\nlink: %s\ndate: %s\n%s\n\n",
			i.Title, i.Link, i.Date, conorsum(i))
	}
}

func conorsum(i *rss.Item) string {
	var s string
	switch {
	case len(i.Content) > 0:
		s = i.Content
	case len(i.Summary) > 0:
		s = i.Summary
	default:
		return ""
	}
	return html.UnescapeString(s)
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
