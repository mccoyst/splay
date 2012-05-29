// Copyright © 2012 Steve McCoy.
// Licensed under the MIT License.
package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// LocateArtist returns a Music object, or an error if none
// can be found which match the given pattern.
//
// Patterns are not patterns in the sense of, say, a regular expression,
// but are literal text which is used to make a best-guess match for
// artists, albums, and songs.
func LocateArtist(pattern string) (Music, error) {
	mloc, err := musicloc()
	if err != nil {
		return nil, err
	}
	artists, err := subDirs(mloc)
	if err != nil {
		return nil, err
	}

	i := find(artists, pattern)
	if i < 0 {
		return nil, nil
	}

	loc := filepath.Join(mloc, artists[i].Name())
	return newArtist(loc), nil
}

// LocateAlbum returns a Music object, or an error if none
// can be found which match the given pattern.
//
// Patterns are not patterns in the sense of, say, a regular expression,
// but are literal text which is used to make a best-guess match for
// artists, albums, and songs.
func LocateAlbum(pattern string) (Music, error) {
	mloc, err := musicloc()
	if err != nil {
		return nil, err
	}
	artists, err := subDirs(mloc)
	if err != nil {
		return nil, err
	}

	allalbums := []os.FileInfo{}
	allnames := []string{}
	for _, artist := range artists {
		aloc := filepath.Join(mloc, artist.Name())
		albums, err := subDirs(aloc)
		if err != nil {
			return nil, err
		}

		allalbums = append(allalbums, albums...)

		for _, album := range albums {
			allnames = append(allnames, filepath.Join(aloc, album.Name()))
		}
	}

	i := find(allalbums, pattern)
	if i < 0 {
		return nil, nil
	}

	return newAlbum(allnames[i], false), nil
}

// musicloc returns the path to the current user's Music folder,
// or an error if it doesn't exist.
func musicloc() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, "Music"), nil
}

// subFiles returns a list of FileInfos for all files under path.
func subFiles(path string) ([]os.FileInfo, error) {
	return contents(path, func(f os.FileInfo) bool {
		return !f.IsDir() && f.Name() != ".DS_Store"
	})
}

// subDirs returns a list of FileInfos for all directories under path.
func subDirs(path string) ([]os.FileInfo, error) {
	return contents(path, func(f os.FileInfo) bool {
		return f.IsDir()
	})
}

// contents returns a list of FileInfos for all acceptable
// entries under the given path.
func contents(path string, accept func(os.FileInfo) bool) ([]os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	allsubs, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}

	subs := make([]os.FileInfo, 0, len(allsubs))
	for _, f := range allsubs {
		if accept(f) {
			subs = append(subs, f)
		}
	}

	return subs, nil
}

// match returns a non-negative score iff s fits the pattern, a negative value
// otherwise. One score is better than another if it has a lower value.
func match(pattern, s string) int {
	s = clean(strings.ToLower(s))
	pattern = clean(strings.ToLower(pattern))
	if !strings.Contains(s, pattern) {
		return -1
	}
	d := len(s) - len(pattern)
	if d < 0 {
		return -d
	}
	return d
}

// clean returns s without any non-alphanumeric runes.
func clean(s string) string {
	buf := new(bytes.Buffer)
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			_, _ = buf.WriteRune(r)
		}
	}
	return buf.String()
}

// The Music interface provides methods for identifying and playing
// the different groupings of music (Artist, Album, Track)
type Music interface {
	Path() string
	Play(string, string, bool) error
	List(string) error
}

// An artist represents all of the albums by an artist.
type artist struct {
	path string
}

func newArtist(path string) Music {
	return &artist{path}
}

func (a *artist) Path() string {
	return a.path
}

func (a *artist) Play(cmd, start string, tracks bool) error {
	return a.doPerAlbum(start, func(album os.FileInfo) error {
		p := filepath.Join(a.Path(), album.Name())
		if err := newAlbum(p, true).Play(cmd, "", tracks); err != nil {
			return err
		}
		return nil
	})
}

func (a *artist) List(start string) error {
	return a.doPerAlbum(start, func(album os.FileInfo) error {
		fmt.Println(album.Name())
		return nil
	})
}

func (a *artist) doPerAlbum(start string, f func(os.FileInfo) error) error {
	albums, err := subDirs(a.Path())
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(int64(time.Now().Second())))
	for i := range albums {
		n := intnRange(r, i, len(albums))
		albums[i], albums[n] = albums[n], albums[i]
	}

	s := find(albums, start)
	if s < 0 {
		return newError("I failed to find an album matching this pattern: %q", start)
	}

	albums = append(albums[s:len(albums)], albums[0:s]...)

	for _, album := range albums {
		if err := f(album); err != nil {
			return err
		}
	}
	return nil
}

// intnRange returns a non-negative int in the range [b,e).
func intnRange(r *rand.Rand, b, e int) int {
	return r.Intn(e-b) + b
}

// An album represents all of the tracks of an album.
type album struct {
	path     string
	showName bool
}

func newAlbum(path string, showName bool) Music {
	return &album{path, showName}
}

func (a *album) Path() string {
	return a.path
}

func (a *album) Play(cmd, start string, tracks bool) error {
	return a.doPerSong(start, func(song os.FileInfo) error {
		if tracks {
			n := trimExt(song.Name())
			if a.showName {
				_, p := filepath.Split(a.Path())
				n = p + "/" + n
			}
			fmt.Println(n)
		}
		f := filepath.Join(a.Path(), song.Name())
		c := exec.Command(cmd, f)
		return c.Run()
	})
}

func (a *album) List(start string) error {
	return a.doPerSong(start, func(song os.FileInfo) error {
		fmt.Println(song.Name())
		return nil
	})
}

func (a *album) doPerSong(start string, f func(os.FileInfo) error) error {
	songs, err := subFiles(a.Path())
	if err != nil {
		return err
	}

	s := find(songs, start)
	if s < 0 {
		return newError("I failed to find a song matching this pattern: %q", start)
	}

	for _, song := range songs[s:] {
		if err := f(song); err != nil {
			return err
		}
	}

	return nil
}

// find returns the index into fi of the acceptable FileInfo matching
// the given pattern, or 0 if not found.
func find(fi []os.FileInfo, pattern string) int {
	if pattern == "" {
		return 0
	}

	best := 9999
	loc := -1
	for i := range fi {
		m := match(pattern, fi[i].Name())
		if m < 0 {
			continue
		}
		if m < best {
			best = m
			loc = i
		}
	}
	return loc
}

type Error struct {
	what string
}

func (e *Error) Error() string {
	return e.what
}

func newError(what string, args ...interface{}) error {
	return &Error{fmt.Sprintf(what, args...)}
}

// trimExt returns s, minus any trailing extension.
// E.g. trimExt("dog.txt.orig") returns "dog.txt".
func trimExt(s string) string {
	return s[0 : len(s)-len(filepath.Ext(s))]
}
