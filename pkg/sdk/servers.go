package sdk

import "fmt"

func (c *Client) ListServers() ([]Server, error) {
	var servers []Server
	err := c.get("/servers", &servers)
	return servers, err
}

func (c *Client) GetServer(id string) (*Server, error) {
	var server Server
	err := c.get("/servers/"+id, &server)
	return &server, err
}

func (c *Client) CreateServer(req CreateServerRequest) error {
	return c.post("/servers", req, nil)
}

func (c *Client) StartServer(id string) error {
	return c.post(fmt.Sprintf("/servers/%s/start", id), nil, nil)
}

func (c *Client) StopServer(id string) error {
	return c.post(fmt.Sprintf("/servers/%s/stop", id), nil, nil)
}

func (c *Client) DeleteServer(id string) error {
	return c.delete(fmt.Sprintf("/servers/%s", id))
}

func (c *Client) GetServerStats() (map[string]ServerStats, error) {
	var stats map[string]ServerStats
	err := c.get("/servers-stats", &stats)
	return stats, err
}

func (c *Client) ListLoaders() ([]string, error) {
	var loaders []string
	err := c.get("/loaders", &loaders)
	return loaders, err
}

func (c *Client) ListLoaderVersions(loader string) ([]string, error) {
	var versions []string
	err := c.get(fmt.Sprintf("/loaders/%s/versions", loader), &versions)
	return versions, err
}

func (c *Client) GetLoaderMetadata(loader string, options LoaderOptions) (*LoaderMetadata, error) {
	path := fmt.Sprintf("/loaders/%s/metadata?mcVersion=%s&includeSnapshots=%t&includeUnstable=%t&buildVersion=%s&loaderVersion=%s&installerVersion=%s",
		loader,
		options.MCVersion,
		options.IncludeSnapshots,
		options.IncludeUnstable,
		options.BuildVersion,
		options.LoaderVersion,
		options.InstallerVersion,
	)
	var md LoaderMetadata
	err := c.get(path, &md)
	return &md, err
}

func (c *Client) CheckUpdates() (*UpdateInfo, error) {
	var info UpdateInfo
	err := c.get("/updates", &info)
	return &info, err
}

func (c *Client) GetPortRange() (*PortRange, error) {
	var pr PortRange
	err := c.get("/settings/port-range", &pr)
	return &pr, err
}

func (c *Client) SetPortRange(start, end int) error {
	payload := map[string]int{
		"start": start,
		"end":   end,
	}
	return c.put("/settings/port-range", payload)
}
