package utils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func EncodeParam(s string) string {
	return url.QueryEscape(s)
}

func ToLowerCase(s string) string {
	var result string
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}

	return result
}

func isPathValid(path string) bool {
	dir, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if !dir.IsDir() {
		return false
	}

	return true
}

func NewDir(path string) (string, error) {
	if !isPathValid(path) {
		return "", errors.New("invalid path")
	}

	dirName := "YourSpotifyMusic"
	fullPath := filepath.Join(path, dirName)

	if runtime.GOOS == "windows" {
		fullPath = fullPath + "\\"
	} else {
		fullPath = fullPath + "/"
	}

	err := os.Mkdir(fullPath, 0700)
	if err != nil {
		fmt.Sprintln("Error: %w", err)
		return "", err
	}

	return fullPath, nil
}

func ToZip(dir, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filePath == dir {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, _ = filepath.Rel(dir, filePath)

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func GetFileSize(filename string) (int64, error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}

	size := int64(fileInfo.Size())
	return size, nil
}

func RemoveAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, e := transform.String(t, s)
	if e != nil {
		panic(e)
	}

	return output
}

func RemoveInvalidChars(input string, invalidChars []byte) string {
	filter := func(r rune) rune {
		for _, c := range invalidChars {
			if byte(r) == c {
				return -1 // remove the char
			}
		}
		return r
	}

	return strings.Map(filter, input)
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

// catch interrupt signal (ctrl+c)
func SetupCloseHandler(tempDir, zipFile string) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		deleteTempFiles(tempDir, zipFile)
		os.Exit(0)
	}()
}

func deleteTempFiles(tempDir, zipFile string) {
	os.RemoveAll(tempDir)
	os.Remove(zipFile)
}
