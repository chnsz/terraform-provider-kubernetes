// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"embed"
	"fmt"
)

//go:embed assets/v2.json
var openApiV2File embed.FS

//go:embed assets/v3.json
var openApiV3File embed.FS

func LoadV2data() ([]byte, error) {
	fileContent, err := openApiV2File.ReadFile("assets/v2.json")
	if err != nil {
		return nil, fmt.Errorf("error reading assets/v2.json: %v", err)
	}
	return fileContent, nil
}

func LoadV3data() ([]byte, error) {
	fileContent, err := openApiV3File.ReadFile("assets/v3.json")
	if err != nil {
		return nil, fmt.Errorf("error reading assets/v3.json: %v", err)
	}
	return fileContent, nil
}
