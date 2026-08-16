package loader

import "naviserver/internal/domain"

type LoaderOptions struct {
	MCVersion        string `json:"mcVersion,omitempty"`
	IncludeSnapshots bool   `json:"includeSnapshots,omitempty"`
	IncludeUnstable  bool   `json:"includeUnstable,omitempty"`
	BuildVersion     string `json:"buildVersion,omitempty"`
	LoaderVersion    string `json:"loaderVersion,omitempty"`
	InstallerVersion string `json:"installerVersion,omitempty"`
	JavaPath         string `json:"-"`
}

type LoaderMetadata struct {
	LatestVersion     string   `json:"latestVersion,omitempty"`
	MinecraftVersions []string `json:"minecraftVersions,omitempty"`
	BuildVersions     []string `json:"buildVersions,omitempty"`
	LoaderVersions    []string `json:"loaderVersions,omitempty"`
	InstallerVersions []string `json:"installerVersions,omitempty"`
}

type ServerLoader interface {
	Load(options LoaderOptions, destDir string, progressChan chan<- domain.ProgressEvent) (string, error)
	GetSupportedVersions(options LoaderOptions) ([]string, error)
	GetMetadata(options LoaderOptions) (*LoaderMetadata, error)
}
