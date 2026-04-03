package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-NaviServer-Client", "CLI")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func (c *Client) get(path string, target interface{}) error {
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("error: %s", msg)
		}
		return fmt.Errorf("API error (%d)", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) post(path string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(bodyBytes))
		if msg != "" {
			return fmt.Errorf("error: %s", msg)
		}
		return fmt.Errorf("API error (%d)", resp.StatusCode)
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

func (c *Client) delete(path string) error {
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("error: %s", msg)
		}
		return fmt.Errorf("API error (%d)", resp.StatusCode)
	}
	return nil
}

func (c *Client) put(path string, body interface{}) error {
	resp, err := c.doRequest(http.MethodPut, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(bodyBytes))
		if msg != "" {
			return fmt.Errorf("Error: %s", msg)
		}
		return fmt.Errorf("API error (%d)", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetWebSocketURL(path string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	u.Scheme = "ws"
	u.Path = path
	return u.String(), nil
}
