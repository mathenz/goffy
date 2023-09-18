package youtube

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/raitonoberu/ytmusic"
	"github.com/tidwall/gjson"

	sp "github.com/mathenz/goffy/spotify"
	"github.com/mathenz/goffy/utils"
)

type TrackMatch struct {
	Id    string
	Ratio float64
}

type YTSong struct {
	Title, Artist, Album, Id string
}

type Ratio struct {
	Title, Artist, Album, Total float64
}

// select the best result on YouTube Music
func Match(spTrack *sp.Track, results []YTSong) string {
	if results == nil {
		return ""
	}

	trackMatch := TrackMatch{}
	ratio := Ratio{}

	spTrack.Title = utils.RemoveAccents(utils.ToLowerCase(spTrack.Title))
	spTrack.Artist = utils.RemoveAccents(utils.ToLowerCase(spTrack.Artist))
	spTrack.Album = utils.RemoveAccents(utils.ToLowerCase(spTrack.Album))

	for _, result := range results {
		if result.Title != "" {
			ratio.Title = strutil.Similarity(result.Title, spTrack.Title, metrics.NewLevenshtein())
		} else {
			continue
		}

		if result.Artist != "" {
			if strings.Contains(result.Artist, spTrack.Artist) {
				ratio.Artist = strutil.Similarity(result.Artist, spTrack.Artist, metrics.NewLevenshtein())
			}
		} else {
			continue
		}

		if result.Album != "" {
			if result.Album == spTrack.Title {
				ratio.Album = 1
			} else {
				ratio.Album = strutil.Similarity(result.Album, spTrack.Album, metrics.NewLevenshtein())
			}
		} else {
			continue
		}

		ratio.Total = (ratio.Title + ratio.Artist + ratio.Album) / 3

		if ratio.Total > trackMatch.Ratio {
			if !strings.Contains(result.Title, spTrack.Title) && !strings.Contains(result.Artist, spTrack.Artist) {
				continue
			} else {
				trackMatch.Id = result.Id
				trackMatch.Ratio = ratio.Total
			}
		}
	}

	return trackMatch.Id
}

// build each YouTube's track and return an slice of them
func (yt YTSong) buildResults(jsonResponse string) []YTSong {
	var YTSongs []YTSong
	jsonResults := gjson.Get(jsonResponse, "tracks").Array()
	limit := 2

	for _, result := range jsonResults {
		if len(YTSongs) >= limit {
			break
		}

		title := result.Get("title").String()
		artist := result.Get("artists.#.name").String()
		album := result.Get("album.name").String()
		id := result.Get("videoId").String()

		item := YTSong{
			Title:  utils.RemoveAccents(utils.ToLowerCase(title)),
			Artist: utils.RemoveAccents(utils.ToLowerCase(artist)),
			Album:  utils.RemoveAccents(utils.ToLowerCase(album)),
			Id:     id,
		}

		YTSongs = append(YTSongs, item)
	}

	return YTSongs
}

func VideoID(spTrack sp.Track) (string, error) {
	YTSong := YTSong{}
	query := fmt.Sprintf("%s %s %s", spTrack.Title, spTrack.Artist, spTrack.Album) // example: "Little Sun Blues Pills Blues Pills"
	search := ytmusic.TrackSearch(query)                                           // "github.com/raitonoberu/ytmusic"

	results, err := search.Next()
	if err != nil {
		return "", err
	}

	jsonStr, _ := json.Marshal(results)
	songsResults := YTSong.buildResults(string(jsonStr))
	id := Match(&spTrack, songsResults)
	return id, nil
}
