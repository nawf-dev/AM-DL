package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"main/internal/support"
	"main/utils/ampapi"

	"github.com/AlecAivazis/survey/v2"
)

type storefrontFallbackDecision struct {
	URL     string
	Message string
}

type scoredSongCandidate struct {
	Data   ampapi.SongRespData
	Score  int
	Reason string
}

type scoredAlbumCandidate struct {
	Data   ampapi.AlbumRespData
	Score  int
	Reason string
}

func resolveStorefrontFallbackURLs(urls []string, token string) ([]string, error) {
	accountStorefront, err := accountStorefrontCode()
	if err != nil || accountStorefront == "" {
		return urls, nil
	}

	resolved := make([]string, 0, len(urls))
	for _, rawURL := range urls {
		decision, err := resolveStorefrontFallbackURL(rawURL, accountStorefront, token)
		if err != nil {
			return nil, err
		}
		if decision.Message != "" {
			fmt.Println(decision.Message)
		}
		resolved = append(resolved, decision.URL)
	}
	return resolved, nil
}

func accountStorefrontCode() (string, error) {
	info, err := fetchWrapperAccountInfo()
	if err != nil {
		return "", err
	}
	code := storefrontCodeFromStorefrontID(info.StorefrontID)
	if code == "" {
		return "", fmt.Errorf("unsupported wrapper storefront_id: %s", info.StorefrontID)
	}
	return code, nil
}

func resolveStorefrontFallbackURL(rawURL, accountStorefront, token string) (storefrontFallbackDecision, error) {
	trimmed := strings.TrimSpace(rawURL)
	if !looksLikeAppleMusicURL(trimmed) {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}

	urlStorefront := requestedStorefront(trimmed)
	if urlStorefront == "" || urlStorefront == accountStorefront {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}

	if strings.Contains(trimmed, "/song/") {
		sourceStorefront, sourceSongID := support.ParseSongURL(trimmed)
		return resolveSongStorefrontFallback(trimmed, sourceStorefront, sourceSongID, accountStorefront, token)
	}
	if strings.Contains(trimmed, "/album/") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return storefrontFallbackDecision{URL: rawURL}, nil
		}
		if songID := parsed.Query().Get("i"); songID != "" {
			return resolveSongStorefrontFallback(trimmed, urlStorefront, songID, accountStorefront, token)
		}
		sourceStorefront, albumID := support.ParseAlbumURL(trimmed)
		return resolveAlbumStorefrontFallback(trimmed, sourceStorefront, albumID, accountStorefront, token)
	}

	return storefrontFallbackDecision{URL: rawURL}, nil
}

func resolveSongStorefrontFallback(rawURL, sourceStorefront, sourceSongID, accountStorefront, token string) (storefrontFallbackDecision, error) {
	if sourceStorefront == "" || sourceSongID == "" {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}
	manifest, err := ampapi.GetSongResp(sourceStorefront, sourceSongID, Config.Language, token)
	if err != nil || len(manifest.Data) == 0 {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}
	source := manifest.Data[0]
	query := strings.TrimSpace(source.Attributes.Name + " " + source.Attributes.ArtistName)
	searchResp, err := ampapi.Search(accountStorefront, query, "songs", Config.Language, token, 10, 0)
	if err != nil || searchResp.Results.Songs == nil || len(searchResp.Results.Songs.Data) == 0 {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s), but no %s storefront song candidates were found. Keeping original URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront), strings.ToUpper(accountStorefront))}, nil
	}

	candidates := scoreSongCandidates(source, searchResp.Results.Songs.Data)
	if len(candidates) == 0 {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s), but no confident song candidates were found. Keeping original URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))}, nil
	}

	top := candidates[0]
	if isHighConfidenceSongMatch(source, candidates) {
		return storefrontFallbackDecision{
			URL:     top.Data.Attributes.URL,
			Message: fmt.Sprintf("Auto-switched song storefront %s → %s using match %q by %s (reason: %s).", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront), top.Data.Attributes.Name, top.Data.Attributes.ArtistName, top.Reason),
		}, nil
	}

	selectedURL, selectedLabel, keepOriginal, err := promptSongFallbackChoice(rawURL, sourceStorefront, accountStorefront, source, candidates)
	if err != nil {
		return storefrontFallbackDecision{}, err
	}
	if keepOriginal {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s). Keeping original song URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))}, nil
	}
	return storefrontFallbackDecision{URL: selectedURL, Message: fmt.Sprintf("Selected fallback song match: %s", selectedLabel)}, nil
}

func resolveAlbumStorefrontFallback(rawURL, sourceStorefront, albumID, accountStorefront, token string) (storefrontFallbackDecision, error) {
	if sourceStorefront == "" || albumID == "" {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}
	resp, err := ampapi.GetAlbumResp(sourceStorefront, albumID, Config.Language, token)
	if err != nil || len(resp.Data) == 0 {
		return storefrontFallbackDecision{URL: rawURL}, nil
	}
	source := resp.Data[0]
	query := strings.TrimSpace(source.Attributes.Name + " " + source.Attributes.ArtistName)
	searchResp, err := ampapi.Search(accountStorefront, query, "albums", Config.Language, token, 10, 0)
	if err != nil || searchResp.Results.Albums == nil || len(searchResp.Results.Albums.Data) == 0 {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s), but no %s storefront album candidates were found. Keeping original URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront), strings.ToUpper(accountStorefront))}, nil
	}

	candidates := scoreAlbumCandidates(source, searchResp.Results.Albums.Data)
	if len(candidates) == 0 {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s), but no confident album candidates were found. Keeping original URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))}, nil
	}

	top := candidates[0]
	if isHighConfidenceAlbumMatch(source, candidates) {
		return storefrontFallbackDecision{
			URL:     top.Data.Attributes.URL,
			Message: fmt.Sprintf("Auto-switched album storefront %s → %s using match %q by %s (reason: %s).", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront), top.Data.Attributes.Name, top.Data.Attributes.ArtistName, top.Reason),
		}, nil
	}

	selectedURL, selectedLabel, keepOriginal, err := promptAlbumFallbackChoice(rawURL, sourceStorefront, accountStorefront, source, candidates)
	if err != nil {
		return storefrontFallbackDecision{}, err
	}
	if keepOriginal {
		return storefrontFallbackDecision{URL: rawURL, Message: fmt.Sprintf("Storefront mismatch detected (%s → %s). Keeping original album URL.", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))}, nil
	}
	return storefrontFallbackDecision{URL: selectedURL, Message: fmt.Sprintf("Selected fallback album match: %s", selectedLabel)}, nil
}

func scoreSongCandidates(source ampapi.SongRespData, candidates []ampapi.SongRespData) []scoredSongCandidate {
	scored := make([]scoredSongCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		score, reason := songMatchScore(source, candidate)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredSongCandidate{Data: candidate, Score: score, Reason: reason})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Data.Attributes.URL < scored[j].Data.Attributes.URL
		}
		return scored[i].Score > scored[j].Score
	})
	return scored
}

func songMatchScore(source, candidate ampapi.SongRespData) (int, string) {
	score := 0
	reasons := []string{}
	if source.Attributes.Isrc != "" && source.Attributes.Isrc == candidate.Attributes.Isrc {
		score += 1000
		reasons = append(reasons, "exact ISRC")
	}
	if normalizedCompare(source.Attributes.Name, candidate.Attributes.Name) {
		score += 120
		reasons = append(reasons, "title match")
	}
	if normalizedCompare(source.Attributes.ArtistName, candidate.Attributes.ArtistName) {
		score += 120
		reasons = append(reasons, "artist match")
	}
	if normalizedCompare(source.Attributes.AlbumName, candidate.Attributes.AlbumName) {
		score += 80
		reasons = append(reasons, "album match")
	}
	if source.Attributes.TrackNumber > 0 && source.Attributes.TrackNumber == candidate.Attributes.TrackNumber {
		score += 20
		reasons = append(reasons, "track number match")
	}
	if source.Attributes.DiscNumber > 0 && source.Attributes.DiscNumber == candidate.Attributes.DiscNumber {
		score += 20
		reasons = append(reasons, "disc number match")
	}
	delta := absInt(source.Attributes.DurationInMillis - candidate.Attributes.DurationInMillis)
	switch {
	case delta <= 1000:
		score += 60
		reasons = append(reasons, "duration ±1s")
	case delta <= 3000:
		score += 35
		reasons = append(reasons, "duration ±3s")
	case delta <= 5000:
		score += 15
		reasons = append(reasons, "duration ±5s")
	}
	if source.Attributes.ContentRating != "" && source.Attributes.ContentRating == candidate.Attributes.ContentRating {
		score += 10
		reasons = append(reasons, "content rating match")
	}
	return score, strings.Join(reasons, ", ")
}

func isHighConfidenceSongMatch(source ampapi.SongRespData, candidates []scoredSongCandidate) bool {
	if len(candidates) == 0 {
		return false
	}
	top := candidates[0]
	secondGap := 1000
	if len(candidates) > 1 {
		secondGap = top.Score - candidates[1].Score
	}
	if source.Attributes.Isrc != "" && source.Attributes.Isrc == top.Data.Attributes.Isrc && secondGap >= 60 {
		return true
	}
	return top.Score >= 320 && secondGap >= 45
}

func scoreAlbumCandidates(source ampapi.AlbumRespData, candidates []ampapi.AlbumRespData) []scoredAlbumCandidate {
	scored := make([]scoredAlbumCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		score, reason := albumMatchScore(source, candidate)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredAlbumCandidate{Data: candidate, Score: score, Reason: reason})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Data.Attributes.URL < scored[j].Data.Attributes.URL
		}
		return scored[i].Score > scored[j].Score
	})
	return scored
}

func albumMatchScore(source, candidate ampapi.AlbumRespData) (int, string) {
	score := 0
	reasons := []string{}
	if source.Attributes.Upc != "" && source.Attributes.Upc == candidate.Attributes.Upc {
		score += 1000
		reasons = append(reasons, "exact UPC")
	}
	if normalizedCompare(source.Attributes.Name, candidate.Attributes.Name) {
		score += 140
		reasons = append(reasons, "album title match")
	}
	if normalizedCompare(source.Attributes.ArtistName, candidate.Attributes.ArtistName) {
		score += 120
		reasons = append(reasons, "artist match")
	}
	if source.Attributes.TrackCount > 0 && source.Attributes.TrackCount == candidate.Attributes.TrackCount {
		score += 35
		reasons = append(reasons, "track count match")
	}
	if source.Attributes.ReleaseDate != "" && source.Attributes.ReleaseDate == candidate.Attributes.ReleaseDate {
		score += 30
		reasons = append(reasons, "release date match")
	} else if len(source.Attributes.ReleaseDate) >= 4 && len(candidate.Attributes.ReleaseDate) >= 4 && source.Attributes.ReleaseDate[:4] == candidate.Attributes.ReleaseDate[:4] {
		score += 10
		reasons = append(reasons, "release year match")
	}
	if source.Attributes.ContentRating != "" && source.Attributes.ContentRating == candidate.Attributes.ContentRating {
		score += 10
		reasons = append(reasons, "content rating match")
	}
	return score, strings.Join(reasons, ", ")
}

func isHighConfidenceAlbumMatch(source ampapi.AlbumRespData, candidates []scoredAlbumCandidate) bool {
	if len(candidates) == 0 {
		return false
	}
	top := candidates[0]
	secondGap := 1000
	if len(candidates) > 1 {
		secondGap = top.Score - candidates[1].Score
	}
	if source.Attributes.Upc != "" && source.Attributes.Upc == top.Data.Attributes.Upc && secondGap >= 60 {
		return true
	}
	return top.Score >= 300 && secondGap >= 45
}

func promptSongFallbackChoice(rawURL, sourceStorefront, accountStorefront string, source ampapi.SongRespData, candidates []scoredSongCandidate) (string, string, bool, error) {
	if !stdinIsInteractive() {
		return "", "", false, fmt.Errorf("storefront mismatch detected (%s → %s) but multiple song candidates were found; rerun interactively to choose a fallback", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))
	}
	options := []string{fmt.Sprintf("Keep original URL (%s storefront)", strings.ToUpper(sourceStorefront))}
	labels := map[string]string{}
	urls := map[string]string{}
	max := len(candidates)
	if max > 5 {
		max = 5
	}
	for i := 0; i < max; i++ {
		candidate := candidates[i]
		label := fmt.Sprintf("%s — %s | %s | score %d", candidate.Data.Attributes.Name, candidate.Data.Attributes.ArtistName, candidate.Reason, candidate.Score)
		options = append(options, label)
		labels[label] = label
		urls[label] = candidate.Data.Attributes.URL
	}
	selected := ""
	message := fmt.Sprintf("Song %q exists in %s, but your account storefront is %s. Choose a fallback:", source.Attributes.Name, strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))
	if err := survey.AskOne(&survey.Select{Message: message, Options: options, PageSize: len(options)}, &selected); err != nil {
		return rawURL, "original URL", true, nil
	}
	if selected == options[0] {
		return rawURL, "original URL", true, nil
	}
	return urls[selected], labels[selected], false, nil
}

func promptAlbumFallbackChoice(rawURL, sourceStorefront, accountStorefront string, source ampapi.AlbumRespData, candidates []scoredAlbumCandidate) (string, string, bool, error) {
	if !stdinIsInteractive() {
		return "", "", false, fmt.Errorf("storefront mismatch detected (%s → %s) but multiple album candidates were found; rerun interactively to choose a fallback", strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))
	}
	options := []string{fmt.Sprintf("Keep original URL (%s storefront)", strings.ToUpper(sourceStorefront))}
	labels := map[string]string{}
	urls := map[string]string{}
	max := len(candidates)
	if max > 5 {
		max = 5
	}
	for i := 0; i < max; i++ {
		candidate := candidates[i]
		label := fmt.Sprintf("%s — %s | %d tracks | %s | score %d", candidate.Data.Attributes.Name, candidate.Data.Attributes.ArtistName, candidate.Data.Attributes.TrackCount, candidate.Reason, candidate.Score)
		options = append(options, label)
		labels[label] = label
		urls[label] = candidate.Data.Attributes.URL
	}
	selected := ""
	message := fmt.Sprintf("Album %q exists in %s, but your account storefront is %s. Choose a fallback:", source.Attributes.Name, strings.ToUpper(sourceStorefront), strings.ToUpper(accountStorefront))
	if err := survey.AskOne(&survey.Select{Message: message, Options: options, PageSize: len(options)}, &selected); err != nil {
		return rawURL, "original URL", true, nil
	}
	if selected == options[0] {
		return rawURL, "original URL", true, nil
	}
	return urls[selected], labels[selected], false, nil
}

func normalizedCompare(a, b string) bool {
	return normalizeMatchString(a) == normalizeMatchString(b)
}

func normalizeMatchString(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
