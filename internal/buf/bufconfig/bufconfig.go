// Copyright 2020-2021 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bufconfig contains the configuration functionality.
package bufconfig

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

const (
	// ExternalConfigFilePath is the default configuration file path.
	ExternalConfigFilePath = "buf.mod"

	// ExternalConfigV1Beta1FilePath is the v1beta1 file path.
	ExternalConfigV1Beta1FilePath = "buf.yaml"

	// V1Version is the v1 version.
	V1Version = "v1"

	// V1Beta1Version is the v1beta1 version.
	V1Beta1Version = "v1beta1"
)

var (
	// All versions are all the versions in order.
	AllVersions = []string{
		V1Beta1Version,
		V1Version,
	}
)

// Config is the user config.
type Config struct {
	Version        string
	ModuleIdentity bufmodule.ModuleIdentity
	Build          *bufmodulebuild.Config
	Breaking       *bufbreaking.Config
	Lint           *buflint.Config
}

// Provider is a provider.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ConfigFilePath.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket) (*Config, error)
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfigForData(ctx context.Context, data []byte) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

// WriteConfig writes an initial configuration file into the bucket.
func WriteConfig(
	ctx context.Context,
	writeBucket storage.WriteBucket,
	options ...WriteConfigOption,
) error {
	return writeConfig(
		ctx,
		writeBucket,
		options...,
	)
}

// WriteConfigOption is an option for WriteConfig.
type WriteConfigOption func(*writeConfigOptions)

// WriteConfigWithModuleIdentity returns a new WriteConfigOption that sets the name of the
// module to the given ModuleIdentity.
//
// The default is to not set the name.
func WriteConfigWithModuleIdentity(moduleIdentity bufmodule.ModuleIdentity) WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.moduleIdentity = moduleIdentity
	}
}

// WriteConfigWithDependencyModuleReferences returns a new WriteConfigOption that sets the
// dependencies of the module.
//
// The default is to not have any dependencies.
//
// If this option is used, WriteConfigWithModuleIdentity must also be used.
func WriteConfigWithDependencyModuleReferences(dependencyModuleReferences ...bufmodule.ModuleReference) WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.dependencyModuleReferences = dependencyModuleReferences
	}
}

// WriteConfigWithDocumentationComments returns a new WriteConfigOption that documents the resulting configuration file.
func WriteConfigWithDocumentationComments() WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.documentationComments = true
	}
}

// WriteConfigWithUncomment returns a new WriteConfigOption that uncomments the resulting configuration file
// options that are commented out by default.
//
// If this option is used, WriteConfigWithDocumentationComments must also be used.
func WriteConfigWithUncomment() WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.uncomment = true
	}
}

// ReadConfig reads the configuration from the OS or an override, if any.
//
// Only use in CLI tools.
func ReadConfig(
	ctx context.Context,
	provider Provider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (*Config, error) {
	return readConfig(
		ctx,
		provider,
		readBucket,
		options...,
	)
}

// ReadConfigOption is an option for ReadConfig.
type ReadConfigOption func(*readConfigOptions)

// ReadConfigWithOverride sets the override.
//
// If override is set, this will first check if the override ends in .json or .yaml, if so,
// this reads the file at this path and uses it. Otherwise, this assumes this is configuration
// data in either JSON or YAML format, and unmarshals it.
//
// If no override is set, this reads ExternalConfigFilePath in the bucket.
func ReadConfigWithOverride(override string) ReadConfigOption {
	return func(readConfigOptions *readConfigOptions) {
		readConfigOptions.override = override
	}
}

// ConfigExists checks if a configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	exists, err := storage.Exists(ctx, readBucket, ExternalConfigFilePath)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	// If the default filename does not exist, fallback to previous versions.
	return storage.Exists(ctx, readBucket, ExternalConfigV1Beta1FilePath)
}

// ExternalConfigV1Beta1 represents the on-disk representation of the Config
// at version v1beta1.
type ExternalConfigV1Beta1 struct {
	Version  string                               `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                               `json:"name,omitempty" yaml:"name,omitempty"`
	Deps     []string                             `json:"deps,omitempty" yaml:"deps,omitempty"`
	Build    bufmodulebuild.ExternalConfigV1Beta1 `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreaking.ExternalConfigV1Beta1    `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflint.ExternalConfigV1Beta1        `json:"lint,omitempty" yaml:"lint,omitempty"`
}

// ExternalConfigV1 represents the on-disk representation of the Config
// at version v1.
type ExternalConfigV1 struct {
	Version  string                          `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                          `json:"name,omitempty" yaml:"name,omitempty"`
	Deps     []string                        `json:"deps,omitempty" yaml:"deps,omitempty"`
	Build    bufmodulebuild.ExternalConfigV1 `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreaking.ExternalConfigV1    `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflint.ExternalConfigV1        `json:"lint,omitempty" yaml:"lint,omitempty"`
}

// ExternalConfigVersion defines the subset of all config
// file versions that is used to determine the configuration version.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
