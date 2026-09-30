package main

// Update check: when it opens (then every 6 hours) Parley asks GitHub for the latest release and,
// if it's newer, offers it in the overlay and the tray menu. Nothing is sent
// except a normal web request; no personal data.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const releasesAPI = "https://api.github.com/repos/Duhlude/parley/releases/latest"

type release struct {
	Version     string // "1.6.0"
	URL         string // release page
	SetupURL    string // Parley-Setup.exe download ("" if the release has none)
	SetupSize   int64
	SetupSHA256 string // hex, from GitHub's asset digest ("" if GitHub gave none)
}

// canInstall reports whether this release can be installed from inside
// Parley: it has an installer whose checksum GitHub published.
func (r release) canInstall() bool { return r.SetupURL != "" && len(r.SetupSHA256) == 64 }

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
		Tag    string `json:"tag_name"`
		URL    string `json:"html_url"`
		Draft  bool   `json:"draft"`
		Pre    bool   `json:"prerelease"`
		Assets []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"` // "sha256:<hex>"
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return release{}, err
	}
	if r.Draft || r.Pre {
		return release{}, errors.New("no stable release")
	}
	out := release{Version: strings.TrimPrefix(r.Tag, "v"), URL: r.URL}
	for _, a := range r.Assets {
		if !strings.EqualFold(a.Name, "Parley-Setup.exe") {
			continue
		}
		// Only GitHub's own download links, unless testing.
		if os.Getenv("PARLEY_UPDATE_URL") == "" && !strings.HasPrefix(a.URL, "https://github.com/Duhlude/parley/releases/download/") {
			continue
		}
		out.SetupURL, out.SetupSize = a.URL, a.Size
		if h, ok := strings.CutPrefix(strings.ToLower(a.Digest), "sha256:"); ok {
			if _, err := hex.DecodeString(h); err == nil && len(h) == 64 {
				out.SetupSHA256 = h
			}
		}
	}
	return out, nil
}

// downloadSetup downloads the release's installer into dir, checks its size
// and SHA-256 against what GitHub published, and returns its path. progress
// gets the percentage done (0-100) now and then.
func downloadSetup(r release, dir string, progress func(int)) (string, error) {
	if !r.canInstall() {
		return "", errors.New("this release can't be installed automatically")
	}
	if r.SetupSize <= 0 || r.SetupSize > 300<<20 {
		return "", errors.New("unexpected installer size")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	req, _ := http.NewRequest("GET", r.SetupURL, nil)
	req.Header.Set("User-Agent", "Parley/"+appVersion)
	c := &http.Client{Timeout: 15 * time.Minute}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}
	final := filepath.Join(dir, "Parley-Setup-"+sanitizeVersion(r.Version)+".exe")
	part := final + ".part"
	f, err := os.Create(part)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	var got int64
	last := -1
	buf := make([]byte, 64<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			got += int64(n)
			if got > r.SetupSize {
				f.Close()
				os.Remove(part)
				return "", errors.New("the download is bigger than expected")
			}
			h.Write(buf[:n])
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(part)
				return "", werr
			}
			if pct := int(got * 100 / r.SetupSize); pct/10 != last/10 && progress != nil {
				last = pct
				progress(pct)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(part)
			return "", rerr
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(part)
		return "", err
	}
	if got != r.SetupSize {
		os.Remove(part)
		return "", fmt.Errorf("the download was cut short (%d of %d bytes)", got, r.SetupSize)
	}
	if sum := hex.EncodeToString(h.Sum(nil)); sum != r.SetupSHA256 {
		os.Remove(part)
		return "", errors.New("the download doesn't match GitHub's checksum")
	}
	os.Remove(final)
	if err := os.Rename(part, final); err != nil {
		return "", err
	}
	return final, nil
}

// sanitizeVersion keeps a version string safe to use in a file name.
func sanitizeVersion(v string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '.' {
			return r
		}
		return -1
	}, v)
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
