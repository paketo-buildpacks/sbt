/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sbt

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/mattn/go-shellwords"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

var (
	DefaultArguments = []string{"universal:packageBin"}
	DefaultTarget    = filepath.Join("target", "universal", "*.zip")
)

type Application struct {
	ApplicationPath  string
	Command          string
	Executor         effect.Executor
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
}

func NewApplication(applicationPath string, command string) (Application, error) {
	l, err := sherpa.NewFileListing(applicationPath)
	if err != nil {
		return Application{}, fmt.Errorf("unable to create file listing for %s\n%w", applicationPath, err)
	}
	expected := map[string][]sherpa.FileEntry{"files": l}

	return Application{
		ApplicationPath:  applicationPath,
		Command:          command,
		Executor:         effect.NewExecutor(),
		LayerContributor: libpak.NewLayerContributor("Compiled Application", expected),
	}, nil
}

func (a Application) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	a.Logger.Body(bard.FormatUserConfig("BP_SCALA_BUILD_ARGUMENTS", "the arguments passed to the build system",
		strings.Join(DefaultArguments, " ")))
	a.Logger.Body(bard.FormatUserConfig("BP_SCALA_BUILT_MODULE", "the module to find application artifact in", "<ROOT>"))
	a.Logger.Body(bard.FormatUserConfig("BP_SCALA_BUILT_ARTIFACT", "the built application artifact", DefaultTarget))

	a.LayerContributor.Logger = a.Logger

	layer, err := a.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		arguments, err := a.ResolveArguments()
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to resolve arguments\n%w", err)
		}

		a.Logger.Bodyf("Executing %s %s", filepath.Base(a.Command), strings.Join(arguments, " "))
		if err := a.Executor.Execute(effect.Execution{
			Command: a.Command,
			Args:    arguments,
			Dir:     a.ApplicationPath,
			Stdout:  a.Logger.InfoWriter(),
			Stderr:  a.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}

		artifact, err := a.ResolveArtifact()
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to resolve artifact\n%w", err)
		}

		in, err := os.Open(artifact)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", artifact, err)
		}
		defer in.Close()

		file := filepath.Join(layer.Path, "application.zip")
		if err := sherpa.CopyFile(in, file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s\n%w", artifact, file, err)
		}

		layer.Cache = true
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute application layer\n%w", err)
	}

	a.Logger.Header("Removing source code")
	cs, err := ioutil.ReadDir(a.ApplicationPath)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to list children of %s\n%w", a.ApplicationPath, err)
	}
	for _, c := range cs {
		file := filepath.Join(a.ApplicationPath, c.Name())
		if err := os.RemoveAll(file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to remove %s\n%w", file, err)
		}
	}

	file := filepath.Join(layer.Path, "application.zip")
	in, err := os.Open(file)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	if err := crush.ExtractZip(in, a.ApplicationPath, 0); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to extract %s\n%w", file, err)
	}

	return layer, nil
}

func (Application) Name() string {
	return "application"
}

func (a Application) ResolveArguments() ([]string, error) {
	var err error
	arguments := DefaultArguments

	if s, ok := os.LookupEnv("BP_SBT_BUILD_ARGUMENTS"); ok {
		arguments, err = shellwords.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("unable to parse arguments from %s\n%w", s, err)
		}
	}

	return arguments, nil
}

func (a Application) ResolveArtifact() (string, error) {
	pattern := DefaultTarget
	if s, ok := os.LookupEnv("BP_SBT_BUILT_MODULE"); ok {
		pattern = filepath.Join(s, pattern)
	}
	if s, ok := os.LookupEnv("BP_SBT_BUILT_ARTIFACT"); ok {
		pattern = s
	}

	file := filepath.Join(a.ApplicationPath, pattern)
	artifacts, err := filepath.Glob(file)
	if err != nil {
		return "", fmt.Errorf("unable to find files with %s\n%w", pattern, err)
	}

	if len(artifacts) != 1 {
		sort.Strings(artifacts)
		return "", fmt.Errorf("unable to find built artifact in %s, candidates: %s", pattern, artifacts)
	}

	return artifacts[0], nil
}
