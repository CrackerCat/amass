// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package amass

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/gocrawl"
	"github.com/PuerkitoBio/goquery"
)

// Archiver - represents all objects that perform Memento web archive searches for domain names
type Archiver interface {
	CheckHistory(subdomain *Subdomain)
}

type memento struct {
	url        string
	subdomains chan *Subdomain
	requests   chan *Subdomain
}

func (m *memento) CheckHistory(subdomain *Subdomain) {
	m.requests <- subdomain
}

func MementoWebArchive(u string, subdomains chan *Subdomain) Archiver {
	m := new(memento)

	m.url = u
	m.subdomains = subdomains
	m.requests = make(chan *Subdomain, 100)

	go m.processRequests()
	return m
}

func (a *Amass) ArchiveItArchive() Archiver {
	return MementoWebArchive("https://wayback.archive-it.org/all", a.Names)
}

func (a *Amass) ArchiveIsArchive() Archiver {
	return MementoWebArchive("http://archive.is", a.Names)
}

func (a *Amass) ArquivoArchive() Archiver {
	return MementoWebArchive("http://arquivo.pt/wayback", a.Names)
}

func (a *Amass) BayerischeArchive() Archiver {
	return MementoWebArchive("http://langzeitarchivierung.bib-bvb.de/wayback", a.Names)
}

func (a *Amass) LibraryCongressArchive() Archiver {
	return MementoWebArchive("http://webarchive.loc.gov/all", a.Names)
}

func (a *Amass) PermaArchive() Archiver {
	return MementoWebArchive("http://perma-archives.org/warc", a.Names)
}

func (a *Amass) UKWebArchive() Archiver {
	return MementoWebArchive("http://www.webarchive.org.uk/wayback/archive", a.Names)
}

func (a *Amass) UKGovArchive() Archiver {
	return MementoWebArchive("http://webarchive.nationalarchives.gov.uk", a.Names)
}

func (a *Amass) WaybackMachineArchive() Archiver {
	return MementoWebArchive("http://web.archive.org/web", a.Names)
}

/* Private functions */

func (m *memento) processRequests() {
	var running int
	var queue []*Subdomain
	done := make(chan int, 10)

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	year := time.Now().Year()
	// Only have up to 10 crawlers running at the same time
	for {
		select {
		case sd := <-m.requests:
			queue = append(queue, sd)
		case <-t.C:
			if running >= 10 || len(queue) <= 0 {
				break
			}

			s := queue[0]
			if len(queue) == 1 {
				queue = []*Subdomain{}
			} else {
				queue = queue[1:]
			}

			go crawl(m.url, strconv.Itoa(year), s, m.subdomains, done, 10*time.Second)
			running++
		case <-done:
			running--
		}
	}
}

type ext struct {
	*gocrawl.DefaultExtender
	domainRE                *regexp.Regexp
	mementoRE               *regexp.Regexp
	filter                  map[string]bool
	flock                   sync.RWMutex
	base, year, sub, domain string
	names                   chan *Subdomain
}

func (e *ext) reducedURL(u *url.URL) string {
	orig := u.String()

	idx := e.mementoRE.FindStringIndex(orig)
	if idx == nil {
		return ""
	}

	i := idx[1]
	return fmt.Sprintf("%s/%s/%s", e.base, e.year, orig[i:])
}

func (e *ext) Log(logFlags gocrawl.LogFlags, msgLevel gocrawl.LogFlags, msg string) {
	return
}

func (e *ext) RequestRobots(ctx *gocrawl.URLContext, robotAgent string) (data []byte, doRequest bool) {
	return nil, false
}

func (e *ext) Filter(ctx *gocrawl.URLContext, isVisited bool) bool {
	if isVisited {
		return false
	}

	u := ctx.URL().String()
	r := e.reducedURL(ctx.URL())

	if !strings.Contains(ctx.URL().Path, e.sub) {
		return false
	}

	e.flock.RLock()
	_, ok := e.filter[r]
	e.flock.RUnlock()

	if ok {
		return false
	}

	if u != r {
		// The more refined version has been requested
		// and will cause the reduced version to be filtered
		e.flock.Lock()
		e.filter[r] = true
		e.flock.Unlock()
	}
	return true
}

func (e *ext) Visit(ctx *gocrawl.URLContext, res *http.Response, doc *goquery.Document) (interface{}, bool) {
	in, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, true
	}

	for _, f := range e.domainRE.FindAllString(string(in), -1) {
		e.names <- &Subdomain{Name: f, Domain: e.domain, Tag: ARCHIVE}
	}
	return nil, true
}

func crawl(base, year string, subdomain *Subdomain, names chan *Subdomain, done chan int, timeout time.Duration) {
	domain := subdomain.Domain
	if domain == "" {
		done <- 1
		return
	}

	ext := &ext{
		DefaultExtender: &gocrawl.DefaultExtender{},
		domainRE:        SubdomainRegex(domain),
		mementoRE:       regexp.MustCompile(base + "/[0-9]+/"),
		filter:          make(map[string]bool), // Filter for not double-checking URLs
		base:            base,
		year:            year,
		sub:             subdomain.Name,
		domain:          domain,
		names:           names,
	}

	// Set custom options
	opts := gocrawl.NewOptions(ext)
	opts.CrawlDelay = 500 * time.Millisecond
	opts.LogFlags = gocrawl.LogError
	opts.SameHostOnly = true
	opts.MaxVisits = 20

	c := gocrawl.NewCrawlerWithOptions(opts)
	go c.Run(fmt.Sprintf("%s/%s/%s", base, year, subdomain.Name))

	<-time.After(timeout)
	c.Stop()
	done <- 1
}
