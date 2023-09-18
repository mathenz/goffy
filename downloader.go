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

// concurrency is not necessary because only one track will be downloaded
func dlTrack(url, path string) error {
	trackInfo, err := TrackInfo(url)
	if err != nil {
		return err
	}

	fmt.Println("Getting track info...")

	track := Track{
		Title:  trackInfo.Title,
		Artist: trackInfo.Artist,
		Album:  trackInfo.Album,
	}

	id, err := VideoID(track)
	if err != nil {
		return err
	}

	track.Title, track.Artist = correctFilename(track.Title, track.Artist)
	fmt.Println("Now, downloading track...")
	err = dlAudio(id, path, track.Title, track.Artist)
	if err != nil {
		yellow.Printf("%s - %s could not be downloaded.\n", track.Title, track.Artist)
		os.Exit(1)
	}

	track.Title, track.Artist = correctFilename(track.Title, track.Artist)
	filePath := fmt.Sprintf("%s%s - %s.m4a", path, track.Title, track.Artist)
	if err := m4aTags(filePath, track); err != nil {
		fmt.Println("Error modifying tags:", err)
	}

	fmt.Println("Done.")
	return nil
}

func dlPlaylist(url, path string) error {
	tracks, err := PlaylistInfo(url)
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	fmt.Println("Now, downloading playlist...")
	err = dlTracks(tracks, path)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func dlFromTxt(file, savePath string) error {
	// first, check if the file is a txt
	ext := filepath.Ext(file)
	if !strings.EqualFold(ext, ".txt") {
		return errors.New("file is not a txt")
	}

	// check if it is empty
	txtSize, _ := GetFileSize(file)
	if txtSize <= 0 {
		return errors.New("file is empty")
	}

	fmt.Println("Getting tracks' info...")
	txt, err := os.Open(file)
	if err != nil {
		return err
	}
	defer txt.Close()

	var wg sync.WaitGroup
	var tracksMutex sync.Mutex
	var tracks []Track
	lines := make(chan string, 100)

	processLines := func() {
		defer wg.Done()
		for line := range lines {
			track, err := TrackInfo(line)
			if err != nil {
				yellow.Printf("(URL: %s) - Error obtaining track information: %v\n", line, err)
				continue
			}

			tracksMutex.Lock()
			tracks = append(tracks, *track)
			tracksMutex.Unlock()
		}
	}

	numWorkers := 10
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go processLines()
	}

	scanner := bufio.NewScanner(txt)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines)
	wg.Wait()

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Println("Tracks' info collected:", len(tracks))
	time.Sleep(1 * time.Second)
	fmt.Println("Now, downloading tracks...")

	err = dlTracks(tracks, savePath)
	if err != nil {
		return err
	}

	return nil
}

// download more than one track using goroutines (useful for downloading playlists and multiple tracks from a .txt file)
func dlTracks(tracks []Track, path string) error {
	var wg sync.WaitGroup
	concurrentDownloads := 5
	downloadQueue := make(chan Track, len(tracks))

	for _, t := range tracks {
		downloadQueue <- t
	}
	close(downloadQueue)

	for i := 0; i < concurrentDownloads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range downloadQueue {
				trackCopy := &Track{
					Title:  t.Title,
					Artist: t.Artist,
					Album:  t.Album,
				}

				id, err := VideoID(*trackCopy)
				if id == "" {
					yellow.Println("Error 1:", trackCopy.Title, "by", trackCopy.Artist, "could not be downloaded.")
					continue
				}

				if err != nil {
					yellow.Println("Error 2:", trackCopy.Title, "by", trackCopy.Artist, "could not be downloaded. ")
					continue
				}

				trackCopy.Title, trackCopy.Artist = correctFilename(trackCopy.Title, trackCopy.Artist)
				err = dlAudio(id, path, trackCopy.Title, trackCopy.Artist)
				if err != nil {
					yellow.Println("Error 3:", trackCopy.Title, "by", trackCopy.Artist, "could not be downloaded.")
					continue
				}
				fmt.Println(fmt.Sprintf("'%s' by '%s' downloaded", trackCopy.Title, trackCopy.Artist))
			}
		}()
	}
	wg.Wait()

	fmt.Println("\nHold on, adding metadata to audio files...")
	time.Sleep(1 * time.Second)

	for i, t := range tracks {
		trackCopy := &Track{
			Title:  t.Title,
			Artist: t.Artist,
			Album:  t.Album,
		}

		trackCopy.Title, trackCopy.Artist = correctFilename(trackCopy.Title, trackCopy.Artist)
		filePath := fmt.Sprintf("%s%s - %s.m4a", path, trackCopy.Title, trackCopy.Artist)

		_, err := os.Stat(filePath)
		if err != nil {
			yellow.Println(trackCopy.Title, "by", trackCopy.Artist)
			continue
		}

		fmt.Printf("%d. Tags added to: %s - %s.m4a\n", i+1, trackCopy.Title, trackCopy.Artist)

		if err := m4aTags(filePath, *trackCopy); err != nil {
			yellow.Printf("Error adding tags to: %s (error: %s)\n", filePath, err)
		}
	}

	return nil
}

// github.com/kkdai/youtube
func dlAudio(id, path, title, artist string) error {
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

	// itag code: 140, container: m4a, content: audio, bitrate: 128k
	// change the FindByItag parameter to 139 if you want smaller files (but with a bitrate of 48k)
	formats := video.Formats.FindByItag(140)

	filename := fmt.Sprintf("%s - %s.m4a", title, artist)
	route := filepath.Join(path, filename)

	// in some cases, when attempting to download the audio
	// using the library github.com/kkdai/youtube,
	// the download fails (and shows the file size as 0 bytes)
	// until the second or third attempt.
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

func m4aTags(file string, track Track) error {
	outputFile := file
	index := strings.Index(outputFile, ".m4a")
	if index != -1 {
		result := outputFile[:index]       // filename but without the '.m4a' extension ('title - artist')
		outputFile = result + "2" + ".m4a" // just a temporary dumb name ('title - artist2')
	}
	cmd := exec.Command(
		"ffmpeg",
		"-i", file,
		"-c", "copy",
		"-metadata", fmt.Sprintf("title=%s", track.Title),
		"-metadata", fmt.Sprintf("artist=%s", track.Artist),
		"-metadata", fmt.Sprintf("album=%s", track.Album),
		"-metadata", fmt.Sprintf("album_artist=%s", track.Artist),
		outputFile, // the file with the nice name
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

// fix some invalid file names (windows is the capricious one)
func correctFilename(title, artist string) (string, string) {
	if runtime.GOOS == "windows" {
		invalidChars := []byte{'<', '>', '<', ':', '"', '\\', '/', '|', '?', '*'}
		for _, invalidChar := range invalidChars {
			if strings.Contains(title, string(invalidChar)) || strings.Contains(artist, string(invalidChar)) {
				title = RemoveInvalidChars(title, invalidChars)
				artist = RemoveInvalidChars(artist, invalidChars)
			}
		}
	} else {
		if strings.Contains(title, "/") || strings.Contains(artist, "/") {
			title = strings.ReplaceAll(title, "/", "\\")
			artist = strings.ReplaceAll(artist, "/", "\\")
		}
	}
	return title, artist
}

type Downloader interface {
	Track(url string, args ...string) error
	Playlist(url string, args ...string) error
	FromTxt(url string, args ...string) error
}

type DesktopDownloader struct{}
type MobileDownloader struct{}

func (dd DesktopDownloader) DDownloader(url string, downloadFunc func(string, string) error, args ...string) error {
	path := args[0]
	sep := string(filepath.Separator) // get the directory separator depending on the OS

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
	var savePath string = "." // temporarily save music to the current route

	// before saving the music, it is necessary to delete the temporary folder and the .zip file (if any)
	tempDir := "YourSpotifyMusic"
	zipFile := "YourSpotifyMusic.zip"

	// check if the temporary folder exists
	if _, err := os.Stat(filepath.Join(savePath, tempDir)); err == nil {
		if err := os.RemoveAll(filepath.Join(savePath, tempDir)); err != nil {
			fmt.Println("Error:", err)
		}
	}

	// check if the .zip file exists
	if _, err := os.Stat(filepath.Join(savePath, zipFile)); err == nil {
		if err := os.Remove(filepath.Join(savePath, zipFile)); err != nil {
			fmt.Println("Error:", err)
		}
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

	// compress the temporary folder
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

		zipFile := filepath.Join(currentDir, "YourSpotifyMusic.zip")

		err = ServeMusic(zipFile)
		if err != nil {
			log.Fatalln(err)
			return err
		}
	}

	return nil
}

func (dd DesktopDownloader) Track(url string, args ...string) error {
	return dd.DDownloader(url, dlTrack, args...)
}

func (dd DesktopDownloader) Playlist(url string, args ...string) error {
	return dd.DDownloader(url, dlPlaylist, args...)
}

func (dd DesktopDownloader) FromTxt(url string, args ...string) error {
	return dd.DDownloader(url, dlFromTxt, args...)
}

func (dm MobileDownloader) Track(url string) error {
	return dm.MDownloader(url, dlTrack)
}

func (dm MobileDownloader) Playlist(url string) error {
	return dm.MDownloader(url, dlPlaylist)
}

func (dm MobileDownloader) FromTxt(url string) error {
	return dm.MDownloader(url, dlFromTxt)
}
