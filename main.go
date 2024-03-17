package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	    
	"github.com/fatih/color"
)

var (
	white     = color.New(color.FgWhite)
	boldWhite = white.Add(color.Bold)
)

var (
	trackF    string
	playlistF string
	albumF    string
	fileF     string
	desktopF  string
	mobileF   bool
)

func main() {
	flag.StringVar(&trackF, "t", "", "Download a single track. Usage: -t URL")
	flag.StringVar(&playlistF, "p", "", "Download an entire playlist. Usage: -p URL")
	flag.StringVar(&albumF, "a", "", "Download an album. Usage: -a URL")
	flag.StringVar(&fileF, "f", "", "Download multiple tracks from a txt file. Usage: -f /PATH/TO/TXT")
	flag.StringVar(&desktopF, "d", "", "Specify the path to save the music locally. Usage: -d /PATH/TO/MUSIC/FOLDER/")
	flag.BoolVar(&mobileF, "m", false, "Save music on your mobile device. Don't have to specify any path. Usage: -m")

    flag.Usage = func() {
    		fmt.Print("Usage: ")
    		boldWhite.Println("goffy [option] [url] [platform] [/path/to/music/folder/]")

    		fmt.Println("If [option] is -f, [url] is /path/to/txt")
    		fmt.Println("If [platform] is -m, [path] is omitted.")

    		fmt.Printf("\nOptions:\n")
    		flag.VisitAll(func(f *flag.Flag) {
    			if f.Name != "d" && f.Name != "m" {
    				fmt.Printf("  -%s	%s\n", f.Name, f.Usage)
    			}
    		})
    		fmt.Printf("\nPlatform:\n")
    		flag.VisitAll(func(f *flag.Flag) {
    			if f.Name == "d" || f.Name == "m" {
    				fmt.Printf("  -%s	%s\n", f.Name, f.Usage)
    			}
    		})
    	}
	flag.Parse()
	
	tempDir := filepath.Join(GetCurrentDir(), "YourMusic")
	zipFile := filepath.Join(GetCurrentDir(), "YourMusic.zip")
	InterruptHandler(tempDir)
	InterruptHandler(zipFile)

	ddl := DesktopDownloader{}
	mdl := MobileDownloader{}

	switch {
	case trackF != "" && desktopF != "":
		ddl.Track(trackF, desktopF)
	case playlistF != "" && desktopF != "":
		ddl.Playlist(playlistF, desktopF)
	case albumF != "" && desktopF != "":
		ddl.Album(albumF, desktopF)
	case fileF != "" && desktopF != "":
		ddl.FromTxt(fileF, desktopF)
	case trackF != "" && mobileF:
		mdl.Track(trackF)
	case playlistF != "" && mobileF:
		mdl.Playlist(playlistF)
	case albumF != "" && mobileF:
		mdl.Album(albumF)
	case fileF != "" && mobileF:
		mdl.FromTxt(fileF)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
