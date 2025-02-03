package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/kkdai/youtube/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var (
	Socks5Proxy   = os.Getenv("SOCKS5_PROXY")
	SourceCodeURL = "https://github.com/Abishnoi69/ytdl-api"
)

// Nueva función para extraer el ID del video
func extractVideoID(url string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:v=|be\/)([\w-]{11})`),
		regexp.MustCompile(`^([\w-]{11})$`),
	}

	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(url)
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// Modificamos el handler de YouTube
func handlerYouTube(w http.ResponseWriter, r *http.Request) ([]map[string]string, error) {
	videoURL := r.URL.Query().Get("url")
	if videoURL == "" {
		return nil, fmt.Errorf("por favor provee una URL de YouTube\nUso: /youtube?url=urlDelVideo")
	}

	// Extraer el ID del video
	videoID := extractVideoID(videoURL)
	if videoID == "" {
		return nil, fmt.Errorf("URL de YouTube inválida")
	}

	ytClient := youtube.Client{}
	if Socks5Proxy != "" {
		proxyURL, _ := url.Parse(Socks5Proxy)
		ytClient = youtube.Client{HTTPClient: &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}}
	}

	var videos []map[string]string

	// Primero intentar como video
	video, err := ytClient.GetVideo(videoID)
	if err == nil {
		streamURL, err := ytClient.GetStreamURL(video, &video.Formats[0])
		if err != nil {
			return nil, fmt.Errorf("error al obtener stream: %v", err)
		}

		return []map[string]string{{
			"ID":          video.ID,
			"author":      video.Author,
			"duration":    video.Duration.String(),
			"thumbnail":   video.Thumbnails[0].URL,
			"description": video.Description,
			"stream_url":  streamURL,
			"title":       video.Title,
			"view_count":  fmt.Sprintf("%d", video.Views),
			"url_type":    "video",
		}}, nil
	}

	// Si falla, intentar como playlist
	playlist, err := ytClient.GetPlaylist(videoID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener contenido: %v", err)
	}

	for _, entry := range playlist.Videos {
		video, err := ytClient.VideoFromPlaylistEntry(entry)
		if err != nil {
			log.Printf("Error con video de playlist: %v", err)
			continue
		}

		streamURL, err := ytClient.GetStreamURL(video, &video.Formats[0])
		if err != nil {
			log.Printf("Error al obtener stream: %v", err)
			continue
		}

		videos = append(videos, map[string]string{
			"ID":          video.ID,
			"author":      video.Author,
			"duration":    video.Duration.String(),
			"thumbnail":   video.Thumbnails[0].URL,
			"description": video.Description,
			"stream_url":  streamURL,
			"title":       video.Title,
			"view_count":  fmt.Sprintf("%d", video.Views),
			"url_type":    "playlist",
		})
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("no se encontraron videos")
	}

	return videos, nil
}

// Resto del código se mantiene igual...
