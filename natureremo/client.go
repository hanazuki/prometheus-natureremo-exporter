package natureremo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	HttpClient  *http.Client
	Endpoint    url.URL
	AccessToken string
}

const (
	DEFAULT_ENDPOINT    = "https://api.nature.global"
	API_PATH_DEVICES    = "1/devices"
	API_PATH_APPLIANCES = "1/appliances"
	USER_AGENT          = "prometheus-natureremo-exporter (+https://github.com/hanazuki/prometheus-natureremo-exporter)"
)

func (c *Client) get(ctx context.Context, path string, v interface{}) error {
	path_url, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	url := c.Endpoint.ResolveReference(path_url)

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", fmt.Sprintf(USER_AGENT))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Accept", "application/json")

	res, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("%s responded with %s", url.String(), res.Status)
	}

	return json.NewDecoder(res.Body).Decode(v)
}

func (c *Client) FetchDevices(ctx context.Context) (devices []Device, err error) {
	err = c.get(ctx, API_PATH_DEVICES, &devices)
	return
}

func (c *Client) FetchAppliances(ctx context.Context) (appliances []Appliance, err error) {
	err = c.get(ctx, API_PATH_APPLIANCES, &appliances)
	return
}
