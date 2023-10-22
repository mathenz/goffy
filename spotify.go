/*
This is an unofficial (rustical and intentionally feature-limited) version of the Spotify Web API.
The reason of not using the official API is because it would be a waste of time
because this program has no features related to users' private data
*/

package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type PlaylistEndpoint struct {
	Limit, Offset, TotalCount, Requests int64
}

type Track struct {
	Title, Artist, Album string
}

const (
	tokenEndpoint       = "https://open.spotify.com/get_access_token?reason=transport&productType=web-player"
	trackInitialPath    = "https://api-partner.spotify.com/pathfinder/v1/query?operationName=getTrack&variables="
	playlistInitialPath = "https://api-partner.spotify.com/pathfinder/v1/query?operationName=fetchPlaylist&variables="
	trackEndPath        = `{"persistedQuery":{"version":1,"sha256Hash":"e101aead6d78faa11d75bec5e36385a07b2f1c4a0420932d374d89ee17c70dd6"}}`
	playlistEndPath     = `{"persistedQuery":{"version":1,"sha256Hash":"b39f62e9b566aa849b1780927de1450f47e02c54abf1e66e513f96e849591e41"}}`
)

func accessToken() (string, error) {
	resp, err := http.Get(tokenEndpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	accessToken := gjson.Get(string(body), "accessToken")
	return accessToken.String(), nil
}

/* requests to playlist/track endpoints */
func request(endpoint string) (int, string, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("error on making the request")
	}

	bearer, err := accessToken()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get access token: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+bearer)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error on getting response: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("error on reading response: %w", err)
	}

	return resp.StatusCode, string(body), nil
}

func getID(url string) string {
	parts := strings.Split(url, "/")
	id := strings.Split(parts[4], "?")[0]
	return id
}

func isValidPattern(url, pattern string) bool {
	match, _ := regexp.MatchString(pattern, url)
	return match
}

func TrackInfo(url string) (*Track, error) {
	trackPattern := `^https:\/\/open\.spotify\.com\/track\/[a-zA-Z0-9]{22}\?si=[a-zA-Z0-9]{16}$`
	if !isValidPattern(url, trackPattern) {
		return nil, errors.New("invalid track")
	}

	id := getID(url)
	endpointQuery := EncodeParam(fmt.Sprintf(`{"uri":"spotify:track:%s"}`, id))
	endpoint := trackInitialPath + endpointQuery + "&extensions=" + EncodeParam(trackEndPath)

	statusCode, jsonResponse, err := request(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error on getting track info: %w", err)
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("received non-200 status code: %d", statusCode)
	}

	track := &Track{
		Title:  gjson.Get(jsonResponse, "data.trackUnion.name").String(),
		Artist: gjson.Get(jsonResponse, "data.trackUnion.firstArtist.items.0.profile.name").String(),
		Album:  gjson.Get(jsonResponse, "data.trackUnion.albumOfTrack.name").String(),
	}

	return track.buildTrack(), nil
}

func PlaylistInfo(url string) ([]Track, error) {
	playlistPattern := `^https:\/\/open\.spotify\.com\/playlist\/[a-zA-Z0-9]{22}\?si=[a-zA-Z0-9]{16}$`
	if !isValidPattern(url, playlistPattern) {
		return nil, errors.New("invalid playlist")
	}

	id := getID(url)
	pConf := PlaylistEndpoint{Limit: 400}
	endpointQuery := EncodeParam(fmt.Sprintf(`{"uri":"spotify:playlist:%s","offset":%d,"limit":%d}`, id, pConf.Offset, pConf.Limit))
	endpoint := playlistInitialPath + endpointQuery + "&extensions=" + EncodeParam(playlistEndPath)

	statusCode, jsonResponse, err := request(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting the tracks of the playlist: %w", err)
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("received non-200 status code: %d", statusCode)
	}

	pConf.TotalCount = gjson.Get(jsonResponse, "data.playlistV2.content.totalCount").Int()
	if pConf.TotalCount < 1 {
		return nil, errors.New("playlist is empty")
	}

	name := gjson.Get(jsonResponse, "data.playlistV2.name").String()
	fmt.Printf("Collecting tracks from the playlist '%s'...\n", name)
	time.Sleep(2 * time.Second)

	pConf.Requests = int64(math.Ceil(float64(pConf.TotalCount) / float64(pConf.Limit)))

	var tracks []Track
	tracks = append(tracks, processPlaylist(jsonResponse)...)

	for i := 1; i < int(pConf.Requests); i++ {
		pConf.pagination()
		endpointQuery = EncodeParam(fmt.Sprintf(`{"uri":"spotify:playlist:%s","offset":%d,"limit":%d}`, id, pConf.Offset, pConf.Limit))
		endpoint = playlistInitialPath + endpointQuery + "&extensions=" + EncodeParam(playlistEndPath)

		statusCode, jsonResponse, err = request(endpoint)
		if err != nil {
			return nil, fmt.Errorf("error getting the tracks of the playlist: %w", err)
		}

		if statusCode != 200 {
			return nil, fmt.Errorf("received non-200 status code: %d", statusCode)
		}

		tracks = append(tracks, processPlaylist(jsonResponse)...)
	}

	fmt.Printf("Tracks collected: %d\n", len(tracks))

	return tracks, nil
}

func (t *Track) buildTrack() *Track {
	track := &Track{
		Title:  t.Title,
		Artist: t.Artist,
		Album:  t.Album,
	}

	return track
}

func (pConf *PlaylistEndpoint) pagination() {
	pConf.Offset = pConf.Offset + pConf.Limit
}

/* constructs each Spotify track from JSON body and returns a slice of tracks */
func processPlaylist(jsonResponse string) []Track {
	items := gjson.Get(jsonResponse, "data.playlistV2.content.items").Array()
	var tracks []Track

	for _, item := range items {
		track := &Track{
			Title:  item.Get("itemV2.data.name").String(),
			Artist: item.Get("itemV2.data.artists.items.0.profile.name").String(),
			Album:  item.Get("itemV2.data.albumOfTrack.name").String(),
		}
		tracks = append(tracks, *track.buildTrack())
	}

	return tracks
}
