package main

// Update check: when it opens (then every 6 hours) Parley asks GitHub for the latest release and,
// if it's newer, offers it in the overlay and the tray menu. Nothing is sent
// except a normal web request; no personal data.

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const releasesAPI = "https://api.github.com/repos/Duhlude/parley/releases/latest"

type release struct {
	Version string // "1.6.0"
	URL     string // release page
}

// latestRelease asks GitHub for the newest published release.
func latestRelease() (release, error) {
	u := releasesAPI
	if e := os.Getenv("PARLEY_UPDATE_URL"); e != "" { // testing only
		u = e
	}
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Parley/"+appVersion)
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return release{}, errors.New(resp.Status)
	}
	var r struct {
		Tag   string `json:"tag_name"`
		URL   string `json:"html_url"`
		Draft bool   `json:"draft"`
		Pre   bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return release{}, err
	}
	if r.Draft || r.Pre {
		return release{}, errors.New("no stable release")
	}
	return release{Version: strings.TrimPrefix(r.Tag, "v"), URL: r.URL}, nil
}

// newerVersion reports whether a is newer than b ("1.10.0" > "1.9.2").
func newerVersion(a, b string) bool {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}
