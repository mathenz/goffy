package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/kkdai/youtube/v2"
)

var yellow = color.New(color.FgYellow)

func dlSingleTrack(url, savePath string) error {
	trackInfo, err := TrackInfo(url)
	if err != nil {
		return err
	}

	fmt.Println("Getting track info...")
	time.Sleep(500 * time.Millisecond)
	track := []Track{*trackInfo}

	fmt.Println("Now, downloading track...")
	err = dlTrack(track, savePath)
	if err != nil {
		return err
	}

	return nil
}

func dlPlaylist(url, savePath string) error {
	tracks, err := PlaylistInfo(url)
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	fmt.Println("Now, downloading playlist...")
	err = dlTrack(tracks, savePath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func dlFromTxt(file, savePath string) error {
	tracks, err := processTxt(file)
	if err != nil {
		return err
	}

	fmt.Println("Now, downloading tracks...")
	err = dlTrack(tracks, savePath)
	if err != nil {
		return err
	}

	return nil
}

func processTxt(file string) ([]Track, error) {
	/* first check if it is a txt */
	ext := filepath.Ext(file)
	if !strings.EqualFold(ext, ".txt") {
		return nil, errors.New("file is not a txt")
	}

	/* check if it is empty */
	txtSize, _ := GetFileSize(file)
	if txtSize <= 0 {
		return nil, errors.New("file is empty")
	}

	fmt.Println("Getting tracks' info...")
	txt, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer txt.Close()

	scanner := bufio.NewScanner(txt)
	var wg sync.WaitGroup
	var tracks []Track

	for scanner.Scan() {
		line := scanner.Text()
		wg.Add(1)
		go func(line string) {
			defer wg.Done()
			track, err := TrackInfo(line)
			if err != nil {
				yellow.Printf("(URL: %s) - Error obtaining track information: %v\n", line, err)
			}
			tracks = append(tracks, *track)
		}(line)
	}

	wg.Wait()
	if err := scanner.Err(); err != nil {
		fmt.Println("Error at reading file:", err)
	}

	fmt.Println("Tracks' info collected:", len(tracks))
	return tracks, nil
}

func dlTrack(tracks []Track, path string) error {
	var wg sync.WaitGroup
	var stop = make(chan struct{}, 7)

	for _, t := range tracks {
		wg.Add(1)
		go func(track Track) {
			defer wg.Done()
			defer func() { <-stop }()
			stop <- struct{}{}

			trackCopy := &Track{
				Title:  track.Title,
				Artist: track.Artist,
				Album:  track.Album,
			}

			id, err := VideoID(*trackCopy)
			if id == "" || err != nil {
				yellow.Println("Error (1):", trackCopy.Title, "by", trackCopy.Artist, "could not be downloaded")
				return
			}

			trackCopy.Title, trackCopy.Artist = correctFilename(trackCopy.Title, trackCopy.Artist)
			err = getAudio(id, path, trackCopy.Title, trackCopy.Artist)
			if err != nil {
				yellow.Println("Error (2):", trackCopy.Title, "by", trackCopy.Artist, "could not be downloaded")
				return
			}

			trackCopy.Title, trackCopy.Artist = correctFilename(trackCopy.Title, trackCopy.Artist)
			filePath := fmt.Sprintf("%s%s - %s.m4a", path, trackCopy.Title, trackCopy.Artist)

			if _, err := os.Stat(filePath); err != nil {
				yellow.Println(trackCopy.Title, "by", trackCopy.Artist)
				return
			}

			if err := addTags(filePath, *trackCopy); err != nil {
				yellow.Println(trackCopy.Title, "by", trackCopy.Artist)
			}

			fmt.Printf("'%s' by '%s' was downloaded\n", track.Title, track.Artist)
		}(t)
	}
	wg.Wait()

	return nil
}

/* github.com/kkdai/youtube */
func getAudio(id, path, title, artist string) error {
	dir, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
	}

	if !dir.IsDir() {
		return errors.New("the path is not valid (not a dir)")
	}

	client := youtube.Client{}
	video, err := client.GetVideo(id)
	if err != nil {
		return err
	}

	/* itag code: 140, container: m4a, content: audio, bitrate: 128k */
	/* change the FindByItag parameter to 139 if you want smaller files (but with a bitrate of 48k) */
	formats := video.Formats.FindByItag(140)

	filename := fmt.Sprintf("%s - %s.m4a", title, artist)
	route := filepath.Join(path, filename)

	/* in some cases, when attempting to download the audio
	using the library github.com/kkdai/youtube,
	the download fails (and shows the file size as 0 bytes)
	until the second or third attempt. */
	var fileSize int64
	for fileSize == 0 {
		stream, _, err := client.GetStream(video, formats)
		if err != nil {
			return err
		}

		file, err := os.Create(route)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, stream)
		if err != nil {
			return err
		}

		fileSize, _ = GetFileSize(route)
	}

	return nil
}

func addTags(file string, track Track) error {
	outputFile := file
	index := strings.Index(outputFile, ".m4a")
	if index != -1 {
		result := outputFile[:index]       /* filename but without the '.m4a' extension ('title - artist') */
		outputFile = result + "2" + ".m4a" /* just a temporary dumb name ('title - artist2') */
	}
	cmd := exec.Command(
		"ffmpeg",
		"-i", file,
		"-c", "copy",
		"-metadata", fmt.Sprintf("title=%s", track.Title),
		"-metadata", fmt.Sprintf("artist=%s", track.Artist),
		"-metadata", fmt.Sprintf("album=%s", track.Album),
		"-metadata", fmt.Sprintf("album_artist=%s", track.Artist),
		outputFile, /* the file with the nice name */
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	if err := os.Rename(outputFile, file); err != nil {
		return err
	}

	return nil
}

/* fixes some invalid file names (windows is the capricious one) */
func correctFilename(title, artist string) (string, string) {
	if runtime.GOOS == "windows" {
		invalidChars := []byte{'<', '>', '<', ':', '"', '\\', '/', '|', '?', '*'}
		for _, invalidChar := range invalidChars {
			title = strings.ReplaceAll(title, string(invalidChar), "")
			artist = strings.ReplaceAll(artist, string(invalidChar), "")
		}
	} else {
		title = strings.ReplaceAll(title, "/", "\\")
		artist = strings.ReplaceAll(artist, "/", "\\")
	}

	return title, artist
}

type Downloader interface {
	Track(url string, savePath ...string) error
	Playlist(url string, savePath ...string) error
	FromTxt(url string, savePath ...string) error
}

type DesktopDownloader struct{}
type MobileDownloader struct{}

func (dd DesktopDownloader) DDownloader(url string, downloadFunc func(string, string) error, args ...string) error {
	path := args[0]
	sep := string(filepath.Separator) /* get the directory separator depending on the OS */

	if lastChar := path[len(path)-1]; lastChar != '/' && lastChar != '\\' {
		path += sep
	}

	err := downloadFunc(url, path)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (dm MobileDownloader) MDownloader(url string, downloadFunc func(string, string) error) error {
	var savePath = "." /* temporarily save music to the current route */

	/* before carrying out the download process, it is necessary to delete temporary files (if any) */
	tempDir := filepath.Join(savePath, "YourMusic")
	if _, err := os.Stat(tempDir); err == nil {
		DeleteTempFiles(tempDir)
	}

	zipFile := filepath.Join(savePath, "YourMusic.zip")
	if _, err := os.Stat(zipFile); err == nil {
		DeleteTempFiles(zipFile)
	}

	path, err := NewDir(savePath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = downloadFunc(url, path)
	if err != nil {
		fmt.Println(err)
		return err
	}

	/* compress the temporary folder */
	err = ToZip(path, zipFile)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		fmt.Printf("\nNow, from your phone device, open a new browser window and go to: %s:8080", GetLocalIP())
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			return err
		}

		zipFile := filepath.Join(currentDir, "YourMusic.zip")

		err = ServeMusic(zipFile)
		if err != nil {
			log.Fatalln(err)
			return err
		}
	}

	return nil
}

func (dd DesktopDownloader) Track(url string, savePath ...string) error {
	return dd.DDownloader(url, dlSingleTrack, savePath...)
}

func (dd DesktopDownloader) Playlist(url string, savePath ...string) error {
	return dd.DDownloader(url, dlPlaylist, savePath...)
}

func (dd DesktopDownloader) FromTxt(file string, savePath ...string) error {
	return dd.DDownloader(file, dlFromTxt, savePath...)
}

func (dm MobileDownloader) Track(url string) error {
	return dm.MDownloader(url, dlSingleTrack)
}

func (dm MobileDownloader) Playlist(url string) error {
	return dm.MDownloader(url, dlPlaylist)
}

func (dm MobileDownloader) FromTxt(file string) error {
	return dm.MDownloader(file, dlFromTxt)
}
