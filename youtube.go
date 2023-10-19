package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/raitonoberu/ytmusic"
	"github.com/tidwall/gjson"
)

type TrackMatch struct {
	Id    string
	Ratio float64
}

type YTResult struct {
	Title, Artist, Album, Id string
}

type Ratio struct {
	Title, Artist, Album, Total float64
}

/* selects the best result on YouTube Music */
func Match(spTrack *Track, results []YTResult) string {
	var trackMatch TrackMatch
	spTrack.Title = RemoveAccents(ToLowerCase(spTrack.Title))
	spTrack.Artist = RemoveAccents(ToLowerCase(spTrack.Artist))
	spTrack.Album = RemoveAccents(ToLowerCase(spTrack.Album))

	for _, result := range results {
		ratio := calculateMatchRatio(spTrack, result)
		if ratio > trackMatch.Ratio && isPartialMatch(result, spTrack) {
			trackMatch.Id = result.Id
			trackMatch.Ratio = ratio
		}
	}

	return trackMatch.Id
}

func calculateMatchRatio(spTrack *Track, result YTResult) float64 {
	var ratio Ratio

	ratio.Title = map[bool]float64{result.Title != "": strutil.Similarity(result.Title, spTrack.Title, metrics.NewLevenshtein()), true: 0}[true]
	ratio.Artist = map[bool]float64{result.Artist != "" && strings.Contains(CleanAndNormalize(result.Artist), CleanAndNormalize(spTrack.Artist)): strutil.Similarity(result.Artist, spTrack.Artist, metrics.NewLevenshtein()), true: 0}[true]
	ratio.Album = map[bool]float64{result.Album == result.Title && result.Album == spTrack.Title: 1, true: strutil.Similarity(result.Album, spTrack.Album, metrics.NewLevenshtein())}[true]
	ratio.Total = (ratio.Title + ratio.Artist + ratio.Album) / 3

	return ratio.Total
}

/* last validation before returning the most precise ID from the Match function */
func isPartialMatch(result YTResult, spTrack *Track) bool {
	ytTitle, spTitle := RemoveAccents(ToLowerCase(result.Title)), RemoveAccents(ToLowerCase(spTrack.Title))
	ytTitleSeparated, spTitleSeparated := strings.Fields(ytTitle), strings.Fields(spTitle)

	for _, spField := range spTitleSeparated {
		for _, ytField := range ytTitleSeparated {
			return strings.Contains(ytField, spField)
		}
	}

	return false
}

/* constructs each YouTube result into a structured track and returns a two-element slice */
func (yt YTResult) buildResults(jsonResponse string) []YTResult {
	var ytResults []YTResult
	jsonResults := gjson.Get(jsonResponse, "tracks").Array()
	limit := 2

	for _, result := range jsonResults {
		if len(ytResults) >= limit {
			break
		}

		title := result.Get("title").String()
		artist := result.Get("artists.#.name").String()
		album := result.Get("album.name").String()
		id := result.Get("videoId").String()

		item := YTResult{
			Title:  RemoveAccents(ToLowerCase(title)),
			Artist: RemoveAccents(ToLowerCase(artist)),
			Album:  RemoveAccents(ToLowerCase(album)),
			Id:     id,
		}

		ytResults = append(ytResults, item)
	}

	return ytResults
}

func VideoID(spTrack Track) (string, error) {
	var ytResult YTResult
	query := fmt.Sprintf("'%s' %s %s", spTrack.Title, spTrack.Artist, spTrack.Album) /* example: 'Lovesong' The Cure Disintegration */
	search := ytmusic.TrackSearch(query)                                             /* github.com/raitonoberu/ytmusic */
	result, err := search.Next()
	if err != nil {
		return "", err
	}

	jsonStr, _ := json.Marshal(result)
	ytResults := ytResult.buildResults(string(jsonStr))
	id := Match(&spTrack, ytResults)

	return id, nil
}
