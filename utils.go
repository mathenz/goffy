package main

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
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func EncodeParam(s string) string {
	return url.QueryEscape(s)
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

func IsTxt(file string) bool {
    parts := strings.Index(file, ".")
    	if parts != -1 {
    		extension := file[parts+1:]
    		if strings.ToLower(extension) != "txt" {
    			return false
    		}
    	}
	return true
}

func NewDir(path string) (string, error) {
	if !isPathValid(path) {
		return "", errors.New("invalid path")
	}

	dirName := "YourMusic"
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

func GetFileSize(file string) (int64, error) {
	fileInfo, err := os.Stat(file)
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
				return -1 /* remove the char */
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

func GetCurrentDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	return workingDir
}

// handle ctrl + c (delete music folder and zip file)
func InterruptHandler(resource string) (err error) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        <-c
        DeleteResource(resource)
		DeleteResource(resource)
        os.Exit(1)
    }()
    
    return err
}

func DeleteResource(resource string) {
	if _, err := os.Stat(resource); err == nil {
		if err := os.RemoveAll(resource); err != nil {
			fmt.Println("Error deleting resource:", err)
		}
	}
}

/* used for the last validation in the Match function */
func ExtractFirstWord(value string) string {
	for i := range value {
		if value[i] == ' ' {
			return value[0:i]
		}
	}
	return value
}

/*
i don't know why, but there are artists who,
due to their name, they add a hyphen
between some words of their names
on one platform and not on the other
*/
func CleanAndNormalize(s string) string {
	cleaned := strings.ReplaceAll(s, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return cleaned
}
