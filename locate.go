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
	artists, err := contents(mloc)
	if err != nil {
		return nil, err
	}

	for i := range artists {
		if !artists[i].IsDir() {
			continue
		}
		if match(pattern, artists[i].Name()) {
			loc := filepath.Join(mloc, artists[i].Name())
			return newArtist(loc), nil
		}
	}

	return nil, nil
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
	artists, err := contents(mloc)
	if err != nil {
		return nil, err
	}

	for i := range artists {
		if !artists[i].IsDir() {
			continue
		}

		aloc := filepath.Join(mloc, artists[i].Name())
		albums, err := contents(aloc)
		if err != nil {
			return nil, err
		}

		for i := range albums {
			if !albums[i].IsDir() || !match(pattern, albums[i].Name()) {
				continue
			}
			loc := filepath.Join(aloc, albums[i].Name())
			return newAlbum(loc, false), nil
		}
	}

	return nil, nil
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

// contents returns a list of FileInfos for all of the given path's
// subdirectories.
func contents(path string) ([]os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	subs, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

// match returns true iff s fits the pattern.
func match(pattern, s string) bool {
	s = clean(strings.ToLower(s))
	pattern = clean(strings.ToLower(pattern))
	return strings.Contains(s, pattern)
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
	albums, err := contents(a.Path())
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(int64(time.Now().Second())))
	for i := range albums {
		n := intnRange(r, i, len(albums))
		a := albums[i]
		albums[i] = albums[n]
		albums[n] = a
	}

	s := find(albums, start)

	all := make([]os.FileInfo, 0, len(albums))
	all = append(all, albums[s:len(albums)]...)
	albums = append(all, albums[0:s]...)

	for _, album := range albums {
		if album.Name() == ".DS_Store" {
			continue
		}
		p := filepath.Join(a.Path(), album.Name())
		if err = newAlbum(p, true).Play(cmd, "", tracks); err != nil {
			return err
		}
	}
	return nil
}

// intnRange returns a non-negative int in the range [b,e).
func intnRange(r *rand.Rand, b, e int) int {
	return r.Intn(e-b) + b
}

func (a *artist) List(start string) error {
	albums, err := contents(a.Path())
	if err != nil {
		return err
	}

	s := find(albums, start)

	all := make([]os.FileInfo, 0, len(albums))
	all = append(all, albums[s:len(albums)]...)
	albums = append(all, albums[0:s]...)

	for _, album := range albums {
		fmt.Println(album.Name())
	}

	return nil
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
	songs, err := contents(a.Path())
	if err != nil {
		return err
	}

	s := find(songs, start)

	for i := s; i < len(songs); i++ {
		if tracks {
			n := trimExt(songs[i].Name())
			if a.showName {
				_, p := filepath.Split(a.Path())
				n = p + "/" + n
			}
			fmt.Println(n)
		}
		f := filepath.Join(a.Path(), songs[i].Name())
		c := exec.Command(cmd, f)
		err = c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *album) List(start string) error {
	songs, err := contents(a.Path())
	if err != nil {
		return err
	}

	s := find(songs, start)

	for i := s; i < len(songs); i++ {
		fmt.Println(songs[i].Name())
	}
	return nil
}

// find returns the index into fi of the given pattern, or 0 if not found.
func find(fi []os.FileInfo, pattern string) int {
	for i := range fi {
		if match(pattern, fi[i].Name()) {
			return i
		}
	}
	return 0
}

type Error struct {
	what string
}

func (e *Error) Error() string {
	return e.what
}

func newError(what string) error {
	return &Error{what}
}

// trimExt returns s, minus any trailing extension.
// E.g. trimExt("dog.txt.orig") returns "dog.txt".
func trimExt(s string) string {
	return s[0 : len(s)-len(filepath.Ext(s))]
}
