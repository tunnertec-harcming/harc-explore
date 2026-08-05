package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/harc/soundscape/apps/speaker/internal/models"
)

type Client struct {
	BaseURL    string
	DeviceID   string
	HTTPClient *http.Client
}

func New(baseURL, deviceID string) *Client {
	return &Client{
		BaseURL:  trimSlash(baseURL),
		DeviceID: deviceID,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func (c *Client) getJSON(path string, dest any) error {
	res, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("GET %s: %s (%s)", path, res.Status, string(b))
	}
	return json.NewDecoder(res.Body).Decode(dest)
}

func (c *Client) postJSON(path string, body any, dest any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := c.HTTPClient.Post(c.BaseURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("POST %s: %s (%s)", path, res.Status, string(raw))
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(dest)
}

func (c *Client) Bootstrap() (models.SpeakerBootstrap, error) {
	var out models.SpeakerBootstrap
	err := c.getJSON("/v1/speaker/bootstrap", &out)
	return out, err
}

func (c *Client) Pack(id string) (models.Pack, error) {
	var out models.Pack
	err := c.getJSON("/packs/"+id, &out)
	return out, err
}

func (c *Client) Manifest(id string) (models.PackManifest, error) {
	var out models.PackManifest
	err := c.getJSON("/packs/"+id+"/manifest", &out)
	return out, err
}

func (c *Client) Download(path string) ([]byte, error) {
	res, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: %s", path, res.Status)
	}
	return io.ReadAll(res.Body)
}

func (c *Client) Recommend(mode models.ModeID, hour int) (models.RecommendResult, error) {
	var out models.RecommendResult
	err := c.postJSON("/v1/evolution/recommend", map[string]any{
		"device_id": c.DeviceID, "mode": mode, "hour_local": hour,
	}, &out)
	return out, err
}

func (c *Client) Personalize(packID string, hour int) (models.PersonalizeResult, error) {
	var out models.PersonalizeResult
	err := c.postJSON("/v1/evolution/personalize", map[string]any{
		"device_id": c.DeviceID, "pack_id": packID, "hour_local": hour,
	}, &out)
	return out, err
}

func (c *Client) PostEvents(sessionID string, events []map[string]any) error {
	return c.postJSON("/v1/evolution/events", map[string]any{
		"device_id": c.DeviceID, "session_id": sessionID, "events": events,
	}, &map[string]any{})
}
