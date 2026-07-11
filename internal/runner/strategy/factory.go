package strategy

import "runtime"

func GetRunner(loaderType string) ServerRunner {
	switch loaderType {
	case "forge", "neoforge":
		return &ForgeRunner{}
	case "paper", "vanilla", "fabric":
		return &VanillaRunner{JarName: "server.jar"}
	case "bedrock":
		return &BedrockRunner{GOOS: runtime.GOOS}
	default:
		return &VanillaRunner{JarName: "server.jar"}
	}
}
