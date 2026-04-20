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

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error (%d)", e.StatusCode)
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
		return newAPIError(resp.StatusCode, body)
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
		return newAPIError(resp.StatusCode, bodyBytes)
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
		return newAPIError(resp.StatusCode, body)
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
		return newAPIError(resp.StatusCode, bodyBytes)
	}
	return nil
}

func newAPIError(statusCode int, body []byte) error {
	return &APIError{
		StatusCode: statusCode,
		Message:    strings.TrimSpace(string(body)),
	}
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
