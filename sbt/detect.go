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
	"github.com/paketo-buildpacks/libpak/bard"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
)

const (
	PlanEntrySBT                   = "sbt"
	PlanEntryJVMApplicationPackage = "jvm-application-package"
	PlanEntryJDK                   = "jdk"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	l := bard.NewLogger(os.Stdout)
	file := filepath.Join(context.Application.Path, "build.sbt")
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		l.Logger.Infof("SKIPPED: build.sbt could not be found in %s", file)
		return libcnb.DetectResult{Pass: false}, nil
	} else if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to determine if %s exists\n%w", file, err)
	}

	return libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryJVMApplicationPackage},
					{Name: PlanEntrySBT},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: PlanEntryJDK},
					{Name: PlanEntrySBT},
				},
			},
		},
	}, nil
}
