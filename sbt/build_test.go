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

package sbt_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/libpak/sbom"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libbs"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/sbt/sbt"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx      libcnb.BuildContext
		sbtBuild sbt.Build
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())
		sbtBuild = sbt.Build{ApplicationFactory: &FakeApplicationFactory{}}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("does not contribute distribution if wrapper exists", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "sbt"), []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		result, err := sbtBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		fi, err := os.Stat(filepath.Join(ctx.Application.Path, "sbt"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode()).To(BeEquivalentTo(0755))

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("cache"))
		Expect(result.Layers[1].Name()).To(Equal("application"))
		Expect(result.Layers[1].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Application.Path, "sbt")))
	})

	it("contributes distribution for API 0.7+", func() {
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "sbt",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []string{"cpe:2.3:a:lightbend:sbt:1.5.5:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/sbt@1.5.5",
				},
			},
		}
		ctx.StackID = "test-stack-id"
		result, err := sbtBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("sbt"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "sbt", "bin", "sbt")))

		Expect(result.BOM.Entries).To(HaveLen(0))
	})
	it("contributes distribution for API <=0.6", func() {
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "sbt",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.StackID = "test-stack-id"
		ctx.Buildpack.API = "0.6"

		result, err := sbtBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("sbt"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "sbt", "bin", "sbt")))

		Expect(result.BOM.Entries).To(HaveLen(1))
		Expect(result.BOM.Entries[0].Name).To(Equal("sbt"))
		Expect(result.BOM.Entries[0].Build).To(BeTrue())
		Expect(result.BOM.Entries[0].Launch).To(BeFalse())
	})
}

type FakeApplicationFactory struct{}

func (f *FakeApplicationFactory) NewApplication(
	_ map[string]interface{},
	_ []string,
	_ libbs.ArtifactResolver,
	_ libbs.Cache,
	command string,
	_ *libcnb.BOM,
	_ string,
	_ sbom.SBOMScanner,
	_ string,
) (libbs.Application, error) {
	return libbs.Application{
		Command: command,
	}, nil
}
