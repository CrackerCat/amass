// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/caffix/amass/amass/internal/utils"
)

const (
	DNSDumpsterSourceString string = "DNSDumpster"
)

func DNSDumpsterQuery(domain, sub string) []string {
	var unique []string

	if domain != sub {
		return unique
	}

	page := utils.GetWebPage("https://dnsdumpster.com/", nil)
	if page == "" {
		return unique
	}

	token := dumpsterGetCSRFToken(page)
	if token == "" {
		return unique
	}

	page = dumpsterPostForm(token, domain)
	if page == "" {
		return unique
	}

	re := utils.SubdomainRegex(domain)
	for _, sd := range re.FindAllString(page, -1) {
		if u := utils.NewUniqueElements(unique, sd); len(u) > 0 {
			unique = append(unique, u...)
		}
	}
	return unique
}

func dumpsterGetCSRFToken(page string) string {
	re := regexp.MustCompile("<input type='hidden' name='csrfmiddlewaretoken' value='([a-zA-Z0-9]*)' />")

	if subs := re.FindStringSubmatch(page); len(subs) == 2 {
		return strings.TrimSpace(subs[1])
	}
	return ""
}

func dumpsterPostForm(token, domain string) string {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:         utils.DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	params := url.Values{
		"csrfmiddlewaretoken": {token},
		"targetip":            {domain},
	}

	req, err := http.NewRequest("POST", "https://dnsdumpster.com/", strings.NewReader(params.Encode()))
	if err != nil {
		return ""
	}
	// The CSRF token needs to be sent as a cookie
	cookie := &http.Cookie{
		Name:   "csrftoken",
		Domain: "dnsdumpster.com",
		Value:  token,
	}
	req.AddCookie(cookie)

	req.Header.Set("User-Agent", utils.USER_AGENT)
	req.Header.Set("Accept", utils.ACCEPT)
	req.Header.Set("Accept-Language", utils.ACCEPT_LANG)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://dnsdumpster.com")
	req.Header.Set("X-CSRF-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	// Now, grab the entire page
	in, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return string(in)
}
