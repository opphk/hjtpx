package captchax

import "time"

type Config struct {
	BaseURL    string
	AppID      string
	Timeout    time.Duration
	RetryTimes int
	APIVersion APIVersion
}

type APIVersion string

const (
	APIVersionV1 APIVersion = "v1"
	APIVersionV2 APIVersion = "v2"
)

func NewConfig(baseURL string) *Config {
	return &Config{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		RetryTimes: 3,
		APIVersion: APIVersionV1,
	}
}

func (c *Config) WithAppID(appID string) *Config {
	c.AppID = appID
	return c
}

func (c *Config) WithTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

func (c *Config) WithRetryTimes(retryTimes int) *Config {
	c.RetryTimes = retryTimes
	return c
}

func (c *Config) WithAPIVersion(version APIVersion) *Config {
	c.APIVersion = version
	return c
}
