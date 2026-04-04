package appconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"main/utils/structs"

	"gopkg.in/yaml.v2"
)

const configFileName = "config.yaml"

func Default() structs.ConfigSet {
	return structs.ConfigSet{
		Storefront:                 "us",
		BackendMode:                "docker",
		DefaultDownloadMode:        "alac",
		MediaUserToken:             "your-media-user-token",
		AuthorizationToken:         "your-authorization-token",
		Language:                   "",
		SaveLrcFile:                false,
		LrcType:                    "lyrics",
		LrcFormat:                  "lrc",
		SaveAnimatedArtwork:        false,
		EmbyAnimatedArtwork:        false,
		EmbedLrc:                   true,
		EmbedCover:                 true,
		SaveArtistCover:            false,
		CoverSize:                  "5000x5000",
		CoverFormat:                "jpg",
		AlacSaveFolder:             "AM-DL downloads",
		AtmosSaveFolder:            "AM-DL-Atmos downloads",
		AacSaveFolder:              "AM-DL-AAC downloads",
		MVSaveFolder:               "AM-DL-MV downloads",
		AlbumFolderFormat:          "{AlbumName}",
		PlaylistFolderFormat:       "{PlaylistName}",
		ArtistFolderFormat:         "{UrlArtistName}",
		SongFileFormat:             "{SongNumer}. {SongName}",
		ExplicitChoice:             "[E]",
		CleanChoice:                "[C]",
		AppleMasterChoice:          "[M]",
		MaxMemoryLimit:             256,
		DecryptM3u8Port:            "127.0.0.1:10020",
		GetM3u8Port:                "127.0.0.1:20020",
		GetM3u8Mode:                "hires",
		GetM3u8FromDevice:          true,
		AacType:                    "aac-lc",
		AlacMax:                    192000,
		AtmosMax:                   2768,
		LimitMax:                   200,
		UseSongInfoForPlaylist:     false,
		DlAlbumcoverForPlaylist:    false,
		MVAudioType:                "atmos",
		MVMax:                      2160,
		ConvertAfterDownload:       false,
		ConvertFormat:              "flac",
		ConvertKeepOriginal:        false,
		ConvertSkipIfSourceMatch:   true,
		FFmpegPath:                 "ffmpeg",
		ConvertExtraArgs:           "",
		ConvertWithMetadata:        true,
		ConvertWarnLossyToLossless: true,
		ConvertSkipLossyToLossless: true,
		ConvertCheckBadALAC:        false,
		ConvertDeleteBadALAC:       false,
	}
}

func Path() string {
	return filepath.Join(".", configFileName)
}

func Load() (structs.ConfigSet, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		return structs.ConfigSet{}, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return structs.ConfigSet{}, err
	}
	ApplyDefaults(&cfg)
	return cfg, nil
}

func LoadOrDefault() structs.ConfigSet {
	cfg, err := Load()
	if err != nil {
		cfg = Default()
	}
	ApplyDefaults(&cfg)
	return cfg
}

func Save(cfg structs.ConfigSet) error {
	ApplyDefaults(&cfg)
	body, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), body, 0644)
}

func ApplyDefaults(cfg *structs.ConfigSet) {
	defaults := Default()

	if len(strings.TrimSpace(cfg.Storefront)) != 2 {
		cfg.Storefront = defaults.Storefront
	}
	if cfg.BackendMode == "" {
		cfg.BackendMode = defaults.BackendMode
	}
	if cfg.DefaultDownloadMode == "" {
		cfg.DefaultDownloadMode = defaults.DefaultDownloadMode
	}
	if cfg.DecryptM3u8Port == "" {
		cfg.DecryptM3u8Port = defaults.DecryptM3u8Port
	}
	if cfg.GetM3u8Port == "" {
		cfg.GetM3u8Port = defaults.GetM3u8Port
	}
	if cfg.AacType == "" {
		cfg.AacType = defaults.AacType
	}
	if cfg.CoverSize == "" {
		cfg.CoverSize = defaults.CoverSize
	}
	if cfg.CoverFormat == "" {
		cfg.CoverFormat = defaults.CoverFormat
	}
	if cfg.AlacSaveFolder == "" {
		cfg.AlacSaveFolder = defaults.AlacSaveFolder
	}
	if cfg.AtmosSaveFolder == "" {
		cfg.AtmosSaveFolder = defaults.AtmosSaveFolder
	}
	if cfg.AacSaveFolder == "" {
		cfg.AacSaveFolder = defaults.AacSaveFolder
	}
	if cfg.MVSaveFolder == "" {
		cfg.MVSaveFolder = defaults.MVSaveFolder
	}
	if cfg.AlbumFolderFormat == "" {
		cfg.AlbumFolderFormat = defaults.AlbumFolderFormat
	}
	if cfg.PlaylistFolderFormat == "" {
		cfg.PlaylistFolderFormat = defaults.PlaylistFolderFormat
	}
	if cfg.ArtistFolderFormat == "" {
		cfg.ArtistFolderFormat = defaults.ArtistFolderFormat
	}
	if cfg.SongFileFormat == "" {
		cfg.SongFileFormat = defaults.SongFileFormat
	}
	if cfg.FFmpegPath == "" {
		cfg.FFmpegPath = defaults.FFmpegPath
	}
	if cfg.GetM3u8Mode == "" {
		cfg.GetM3u8Mode = defaults.GetM3u8Mode
	}
	if cfg.LrcType == "" {
		cfg.LrcType = defaults.LrcType
	}
	if cfg.LrcFormat == "" {
		cfg.LrcFormat = defaults.LrcFormat
	}
	if cfg.MaxMemoryLimit == 0 {
		cfg.MaxMemoryLimit = defaults.MaxMemoryLimit
	}
	if cfg.AlacMax == 0 {
		cfg.AlacMax = defaults.AlacMax
	}
	if cfg.AtmosMax == 0 {
		cfg.AtmosMax = defaults.AtmosMax
	}
	if cfg.MVMax == 0 {
		cfg.MVMax = defaults.MVMax
	}
	if cfg.LimitMax == 0 {
		cfg.LimitMax = defaults.LimitMax
	}
	if cfg.AuthorizationToken == "" {
		cfg.AuthorizationToken = defaults.AuthorizationToken
	}
	if cfg.MediaUserToken == "" {
		cfg.MediaUserToken = defaults.MediaUserToken
	}
}

func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

func Validate() error {
	if !Exists() {
		return errors.New("config.yaml not found")
	}
	_, err := Load()
	return err
}
