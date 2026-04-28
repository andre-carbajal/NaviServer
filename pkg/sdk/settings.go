package sdk

func (c *Client) GetLogBufferSize() (*LogBufferSettings, error) {
	var settings LogBufferSettings
	err := c.get("/settings/log-buffer-size", &settings)
	return &settings, err
}

func (c *Client) SetLogBufferSize(size int) error {
	payload := LogBufferSettings{LogBufferSize: size}
	return c.put("/settings/log-buffer-size", payload)
}

func (c *Client) GetPublicIP() (*PublicAddressSettings, error) {
	var settings PublicAddressSettings
	err := c.get("/settings/public-ip", &settings)
	return &settings, err
}

func (c *Client) SetPublicIP(publicIP string) error {
	payload := PublicAddressSettings{PublicIP: publicIP}
	return c.put("/settings/public-ip", payload)
}

func (c *Client) GetNetworkInterfaces() (*NetworkInterfaces, error) {
	var response NetworkInterfaces
	err := c.get("/system/interfaces", &response)
	return &response, err
}
