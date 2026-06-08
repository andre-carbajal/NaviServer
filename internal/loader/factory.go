package loader

import "fmt"

type loaderFactory func() ServerLoader

var loaderFactories = map[string]loaderFactory{
	"vanilla":  func() ServerLoader { return NewVanillaLoader() },
	"paper":    func() ServerLoader { return NewPaperLoader() },
	"fabric":   func() ServerLoader { return NewFabricLoader() },
	"forge":    func() ServerLoader { return NewForgeLoader() },
	"neoforge": func() ServerLoader { return NewNeoForgeLoader() },
}

func RegisterLoaderForTest(loaderType string, factory loaderFactory) func() {
	previous, hadPrevious := loaderFactories[loaderType]
	loaderFactories[loaderType] = factory
	return func() {
		if hadPrevious {
			loaderFactories[loaderType] = previous
			return
		}
		delete(loaderFactories, loaderType)
	}
}

func GetLoader(loaderType string) (ServerLoader, error) {
	factory, ok := loaderFactories[loaderType]
	if !ok {
		return nil, fmt.Errorf("loader type '%s' not supported", loaderType)
	}
	return factory(), nil
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
