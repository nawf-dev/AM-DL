package searchflow

import (
	"fmt"
	"strings"

	"main/utils/ampapi"
	"main/utils/structs"

	"github.com/AlecAivazis/survey/v2"
)

type Result struct {
	URL         string
	Mode        string
	SingleSong  bool
	Cancelled   bool
	DisplayName string
}

type searchResultItem struct {
	Type string
	Name string
	URL  string
	ID   string
}

type qualityOption struct {
	ID          string
	Description string
}

func Handle(cfg structs.ConfigSet, searchType string, queryParts []string, token string) (Result, error) {
	query := strings.Join(queryParts, " ")
	validTypes := map[string]bool{"album": true, "song": true, "artist": true}
	if !validTypes[searchType] {
		return Result{}, fmt.Errorf("invalid search type: %s. Use 'album', 'song', or 'artist'", searchType)
	}

	fmt.Printf("Searching for %ss: %q in storefront %q\n", searchType, query, cfg.Storefront)

	offset := 0
	limit := 15
	apiSearchType := searchType + "s"

	for {
		searchResp, err := ampapi.Search(cfg.Storefront, query, apiSearchType, cfg.Language, token, limit, offset)
		if err != nil {
			return Result{}, fmt.Errorf("error fetching search results: %w", err)
		}

		var items []searchResultItem
		var displayOptions []string
		hasNext := false

		const prevPageOpt = "⬅️  Previous Page"
		const nextPageOpt = "➡️  Next Page"

		if offset > 0 {
			displayOptions = append(displayOptions, prevPageOpt)
		}

		switch searchType {
		case "album":
			if searchResp.Results.Albums != nil {
				for _, item := range searchResp.Results.Albums.Data {
					year := ""
					if len(item.Attributes.ReleaseDate) >= 4 {
						year = item.Attributes.ReleaseDate[:4]
					}
					trackInfo := fmt.Sprintf("%d tracks", item.Attributes.TrackCount)
					detail := fmt.Sprintf("%s (%s, %s)", item.Attributes.ArtistName, year, trackInfo)
					displayOptions = append(displayOptions, fmt.Sprintf("%s - %s", item.Attributes.Name, detail))
					items = append(items, searchResultItem{Type: "Album", Name: item.Attributes.Name, URL: item.Attributes.URL, ID: item.ID})
				}
				hasNext = searchResp.Results.Albums.Next != ""
			}
		case "song":
			if searchResp.Results.Songs != nil {
				for _, item := range searchResp.Results.Songs.Data {
					detail := fmt.Sprintf("%s (%s)", item.Attributes.ArtistName, item.Attributes.AlbumName)
					displayOptions = append(displayOptions, fmt.Sprintf("%s - %s", item.Attributes.Name, detail))
					items = append(items, searchResultItem{Type: "Song", Name: item.Attributes.Name, URL: item.Attributes.URL, ID: item.ID})
				}
				hasNext = searchResp.Results.Songs.Next != ""
			}
		case "artist":
			if searchResp.Results.Artists != nil {
				for _, item := range searchResp.Results.Artists.Data {
					detail := ""
					if len(item.Attributes.GenreNames) > 0 {
						detail = strings.Join(item.Attributes.GenreNames, ", ")
					}
					displayOptions = append(displayOptions, fmt.Sprintf("%s (%s)", item.Attributes.Name, detail))
					items = append(items, searchResultItem{Type: "Artist", Name: item.Attributes.Name, URL: item.Attributes.URL, ID: item.ID})
				}
				hasNext = searchResp.Results.Artists.Next != ""
			}
		}

		if len(items) == 0 && offset == 0 {
			fmt.Println("No results found.")
			return Result{Cancelled: true}, nil
		}

		if hasNext {
			displayOptions = append(displayOptions, nextPageOpt)
		}

		selectedIndex := 0
		err = survey.AskOne(&survey.Select{
			Message:  "Use arrow keys to navigate, Enter to select:",
			Options:  displayOptions,
			PageSize: limit,
		}, &selectedIndex)
		if err != nil {
			return Result{Cancelled: true}, nil
		}

		selectedOption := displayOptions[selectedIndex]
		if selectedOption == nextPageOpt {
			offset += limit
			continue
		}
		if selectedOption == prevPageOpt {
			offset -= limit
			continue
		}

		itemIndex := selectedIndex
		if offset > 0 {
			itemIndex--
		}

		selectedItem := items[itemIndex]
		quality, err := promptForQuality(selectedItem)
		if err != nil {
			return Result{}, fmt.Errorf("could not process quality selection: %w", err)
		}
		if quality == "" {
			fmt.Println("Selection cancelled.")
			return Result{Cancelled: true}, nil
		}

		return Result{
			URL:         selectedItem.URL,
			Mode:        quality,
			SingleSong:  selectedItem.Type == "Song",
			DisplayName: selectedItem.Name,
		}, nil
	}
}

func promptForQuality(item searchResultItem) (string, error) {
	if item.Type == "Artist" {
		fmt.Println("Artist selected. Proceeding to list all albums/videos.")
		return "default", nil
	}

	fmt.Printf("\nFetching available qualities for: %s\n", item.Name)

	qualities := []qualityOption{
		{ID: "alac", Description: "Lossless (ALAC)"},
		{ID: "aac", Description: "High-Quality (AAC)"},
		{ID: "atmos", Description: "Dolby Atmos"},
	}
	qualityOptions := make([]string, 0, len(qualities))
	for _, q := range qualities {
		qualityOptions = append(qualityOptions, q.Description)
	}

	selectedIndex := 0
	err := survey.AskOne(&survey.Select{
		Message:  "Select a quality to download:",
		Options:  qualityOptions,
		PageSize: 5,
	}, &selectedIndex)
	if err != nil {
		return "", nil
	}

	return qualities[selectedIndex].ID, nil
}
