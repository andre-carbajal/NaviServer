package loader

import "fmt"

func GetLoader(loaderType string) (ServerLoader, error) {
	switch loaderType {
	case "vanilla":
		return NewVanillaLoader(), nil
	case "paper":
		return NewPaperLoader(), nil
	case "fabric":
		return NewFabricLoader(), nil
	case "forge":
		return NewForgeLoader(), nil
	case "neoforge":
		return NewNeoForgeLoader(), nil
	default:
		return nil, fmt.Errorf("loader type '%s' not supported", loaderType)
	}
}

func GetLoaderVersions(loaderType string, options LoaderOptions) ([]string, error) {
	loader, err := GetLoader(loaderType)
	if err != nil {
		return nil, err
	}
	return loader.GetSupportedVersions(options)
}

func GetLoaderMetadata(loaderType string, options LoaderOptions) (*LoaderMetadata, error) {
	loader, err := GetLoader(loaderType)
	if err != nil {
		return nil, err
	}
	return loader.GetMetadata(options)
}

func GetAvailableLoaders() []string {
	return []string{"vanilla", "paper", "fabric", "forge", "neoforge"}
}
