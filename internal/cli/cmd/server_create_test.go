package cmd

import "testing"

func TestValidateCreateLoaderFlagsAllowsBedrockPreviews(t *testing.T) {
	previousLoader := createLoader
	previousSnapshots := createIncludeSnapshots
	previousUnstable := createIncludeUnstable
	previousBuild := createBuildVersion
	previousLoaderVersion := createLoaderVersion
	defer func() {
		createLoader = previousLoader
		createIncludeSnapshots = previousSnapshots
		createIncludeUnstable = previousUnstable
		createBuildVersion = previousBuild
		createLoaderVersion = previousLoaderVersion
	}()

	createLoader = "bedrock"
	createIncludeSnapshots = true
	createIncludeUnstable = false
	createBuildVersion = ""
	createLoaderVersion = ""
	if err := validateCreateLoaderFlags(); err != nil {
		t.Fatalf("Bedrock previews should be accepted: %v", err)
	}
}
