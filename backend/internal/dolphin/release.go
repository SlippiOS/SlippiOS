package dolphin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type release struct {
	Draft   bool    `json:"draft"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func get(url string, timeout time.Duration) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		response.Body.Close()
		return nil, fmt.Errorf("%s returned %s", url, response.Status)
	}
	return response, nil
}

func LatestRelease() (string, Asset, error) {
	response, err := get(API, 20*time.Second)
	if err != nil {
		return "", Asset{}, err
	}
	defer response.Body.Close()
	var releases []release
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", Asset{}, fmt.Errorf("cannot decode releases: %w", err)
	}
	for _, entry := range releases {
		if entry.Draft {
			continue
		}
		for _, asset := range entry.Assets {
			if asset.Name != assetName {
				continue
			}
			if !tagPattern.MatchString(entry.TagName) {
				return "", Asset{}, fmt.Errorf("unexpected release tag %q", entry.TagName)
			}
			return entry.TagName, asset, nil
		}
	}
	return "", Asset{}, errors.New("no Slippi release with a Linux AppImage found")
}
