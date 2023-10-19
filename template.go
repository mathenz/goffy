/* html and stuff */

package main

import (
	"fmt"
	"net/http"
)

func ServeMusic(zipFile string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>goffy</title>
		</head>
		<body>
			<div style="text-align: center;">
				<h1 style="font-family:Serif; color:rgb(37, 62, 55);">goffy</h1>
				<p style="font-family:Serif; font-size:15px;">Just click the button to download and enjoy your music.</p>
				<a style="font-family:Serif; font-size:15px; color:rgb(75, 119, 106);" href="%s">YourMusic</a>
			</div>
		</body>
		</html>
		`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, html, zipFile)
	})

	http.HandleFunc(zipFile, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFile)
	})

	http.ListenAndServe(":8080", nil)
	return nil
}
