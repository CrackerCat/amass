// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"bytes"
	"io"
	"log"
	"testing"
)

func TestThreatCrowdQuery(t *testing.T) {
	var b bytes.Buffer
	wr := io.Writer(&b)
	l := log.New(wr, "", log.Lmicroseconds)

	names := ThreatCrowdQuery(testDomain, testDomain, l)

	if len(names) <= 0 {
		t.Errorf("ThreatCrowdQuery did not find any subdomains: %s", b)
	}
}
