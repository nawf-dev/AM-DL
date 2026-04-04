package support

import (
	"os"
	"regexp"
)

var (
	albumURLPattern      = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music|classical\.music)\.apple\.com\/(\w{2})(?:\/album|\/album\/.+))\/(?:id)?(\d[^\D]+)(?:$|\?)`)
	musicVideoURLPattern = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music)\.apple\.com\/(\w{2})(?:\/music-video|\/music-video\/.+))\/(?:id)?(\d[^\D]+)(?:$|\?)`)
	songURLPattern       = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music|classical\.music)\.apple\.com\/(\w{2})(?:\/song|\/song\/.+))\/(?:id)?(\d[^\D]+)(?:$|\?)`)
	playlistURLPattern   = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music|classical\.music)\.apple\.com\/(\w{2})(?:\/playlist|\/playlist\/.+))\/(?:id)?(pl\.[\w-]+)(?:$|\?)`)
	stationURLPattern    = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music)\.apple\.com\/(\w{2})(?:\/station|\/station\/.+))\/(?:id)?(ra\.[\w-]+)(?:$|\?)`)
	artistURLPattern     = regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music|classical\.music)\.apple\.com\/(\w{2})(?:\/artist|\/artist\/.+))\/(?:id)?(\d[^\D]+)(?:$|\?)`)
)

func LimitString(s string, max int) string {
	if max <= 0 {
		return s
	}
	if len([]rune(s)) > max {
		return string([]rune(s)[:max])
	}
	return s
}

func FileExists(path string) (bool, error) {
	f, err := os.Stat(path)
	if err == nil {
		return !f.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ParseAlbumURL(url string) (string, string) {
	return parseURL(albumURLPattern, url)
}

func ParseMusicVideoURL(url string) (string, string) {
	return parseURL(musicVideoURLPattern, url)
}

func ParseSongURL(url string) (string, string) {
	return parseURL(songURLPattern, url)
}

func ParsePlaylistURL(url string) (string, string) {
	return parseURL(playlistURLPattern, url)
}

func ParseStationURL(url string) (string, string) {
	return parseURL(stationURLPattern, url)
}

func ParseArtistURL(url string) (string, string) {
	return parseURL(artistURLPattern, url)
}

func parseURL(pattern *regexp.Regexp, raw string) (string, string) {
	matches := pattern.FindAllStringSubmatch(raw, -1)
	if matches == nil {
		return "", ""
	}
	return matches[0][1], matches[0][2]
}
