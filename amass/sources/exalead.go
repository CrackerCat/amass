// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"fmt"
	"log"

	"github.com/caffix/amass/amass/internal/utils"
)

const (
	ExaleadSourceString string = "Exalead"
)

func ExaleadQuery(domain, sub string, l *log.Logger) []string {
	var unique []string

	if domain != sub {
		return unique
	}

	url := exaleadURL(domain)
	page, err := utils.GetWebPage(url, nil)
	if err != nil {
		l.Printf("Exalead error: %s: %v", url, err)
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

func exaleadURL(domain string) string {
	base := "http://www.exalead.com/search/web/results/"
	format := base + "?q=site:%s+-www?elements_per_page=50"

	return fmt.Sprintf(format, domain)
}
