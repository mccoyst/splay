// © 2012 Steve McCoy. Available under the MIT License.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var byartist = flag.Bool("artist", true, "Prefer artist name matches")
var byalbum = flag.Bool("album", false, "Prefer album name matches")
var start = flag.String("from", "", "The album or track to start playing from")
var list = flag.Bool("list", false, "Print the playlist instead of playing it")
var tracks = flag.Bool("tracks", false, "Print the name of each track before it is played")

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Please provide the name of the thing to play.")
		os.Exit(1)
	}

	pattern := strings.Join(flag.Args(), " ")
	m, err := locate(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if m == nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to find %q\n", pattern)
		os.Exit(1)
	}

	if *list {
		err = m.List(*start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	err = m.Play(*start, *tracks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func locate(pattern string) (Music, error) {
	if *byartist && !*byalbum {
		m, err := LocateArtist(pattern)
		if err != nil {
			return nil, err
		}
		if m != nil {
			return m, nil
		}
	}

	return LocateAlbum(pattern)
}
