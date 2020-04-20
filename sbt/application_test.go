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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/paketo-buildpacks/sbt/sbt"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testApplication(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		cachePath   string
		ctx         libcnb.BuildContext
		application sbt.Application
		executor    *mocks.Executor
		plan        *libcnb.BuildpackPlan
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "application-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "application-layers")
		Expect(err).NotTo(HaveOccurred())

		cachePath, err = ioutil.TempDir("", "application-cache")
		Expect(err).NotTo(HaveOccurred())

		plan = &libcnb.BuildpackPlan{}

		application, err = sbt.NewApplication(ctx.Application.Path, cachePath, "test-command", plan)
		Expect(err).NotTo(HaveOccurred())

		executor = &mocks.Executor{}
		application.Executor = executor
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(cachePath)).To(Succeed())
	})

	it("contributes layer", func() {
		in, err := os.Open(filepath.Join("testdata", "stub-application.zip"))
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target", "universal"), 0755)).To(Succeed())
		out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", "universal", "stub-application.zip"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		Expect(err).NotTo(HaveOccurred())
		_, err = io.Copy(out, in)
		Expect(err).NotTo(HaveOccurred())
		Expect(in.Close()).To(Succeed())
		Expect(out.Close()).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(cachePath, "test-file-1.1.1.jar"), []byte{}, 0644)).To(Succeed())

		application.Logger = bard.NewLogger(ioutil.Discard)
		executor.On("Execute", mock.Anything).Return(nil)

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = application.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Cache).To(BeTrue())

		e := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(e.Command).To(Equal("test-command"))
		Expect(e.Args).To(Equal([]string{"universal:packageBin"}))
		Expect(e.Dir).To(Equal(ctx.Application.Path))
		Expect(e.Stdout).NotTo(BeNil())
		Expect(e.Stderr).NotTo(BeNil())

		Expect(filepath.Join(layer.Path, "application.zip")).To(BeARegularFile())
		Expect(filepath.Join(ctx.Application.Path, "target", "universal", "stub-application.zip")).NotTo(BeAnExistingFile())
		Expect(filepath.Join(ctx.Application.Path, "fixture-marker")).To(BeARegularFile())

		Expect(plan).To(Equal(&libcnb.BuildpackPlan{
			Entries: []libcnb.BuildpackPlanEntry{
				{
					Name: "maven",
					Metadata: map[string]interface{}{
						"dependencies": []libjvm.MavenJAR{
							{
								Name:    "test-file",
								Version: "1.1.1",
								SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
							},
						},
					},
				},
			},
		}))
	})

	context("ResolveArguments", func() {
		it("uses default arguments", func() {
			Expect(application.ResolveArguments()).To(Equal([]string{"universal:packageBin"}))
		})

		context("$BP_SBT_BUILD_ARGUMENTS", func() {

			it.Before(func() {
				Expect(os.Setenv("BP_SBT_BUILD_ARGUMENTS", "test configured arguments")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_SBT_BUILD_ARGUMENTS")).To(Succeed())
			})

			it("parses value from $BP_SBT_BUILD_ARGUMENTS", func() {
				Expect(application.ResolveArguments()).To(Equal([]string{"test", "configured", "arguments"}))
			})
		})
	})

	context("ResolveArtifact", func() {
		it("fails with no files", func() {
			_, err := application.ResolveArtifact()
			Expect(err).To(MatchError("unable to find built artifact in target/universal/*.zip, candidates: []"))
		})

		it("fails with multiple candidates", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target", "universal"), 0755)).To(Succeed())

			for _, f := range []string{"stub-application-1.zip", "stub-application-2.zip", "stub-application-3.zip"} {
				in, err := os.Open(filepath.Join("testdata", "stub-application.zip"))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", "universal", f), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())
			}

			_, err := application.ResolveArtifact()
			Expect(err).To(MatchError(
				fmt.Sprintf("unable to find built artifact in target/universal/*.zip, candidates: [%s %s %s]",
					filepath.Join(ctx.Application.Path, "target", "universal", "stub-application-1.zip"),
					filepath.Join(ctx.Application.Path, "target", "universal", "stub-application-2.zip"),
					filepath.Join(ctx.Application.Path, "target", "universal", "stub-application-3.zip"))))

		})

		it("passes with a single candidate", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target", "universal"), 0755)).To(Succeed())

			in, err := os.Open(filepath.Join("testdata", "stub-application.zip"))
			Expect(err).NotTo(HaveOccurred())

			out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", "universal", "stub-application.zip"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())

			Expect(in.Close()).To(Succeed())
			Expect(out.Close()).To(Succeed())

			Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "target", "universal", "stub-application.zip")))
		})

		context("$BP_SBT_BUILT_MODULE", func() {

			it.Before(func() {
				Expect(os.Setenv("BP_SBT_BUILT_MODULE", "test-directory")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_SBT_BUILT_MODULE")).To(Succeed())
			})

			it("passes with $BP_SBT_BUILT_MODULE", func() {
				Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "test-directory", "target", "universal"), 0755)).To(Succeed())

				in, err := os.Open(filepath.Join("testdata", "stub-application.zip"))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "test-directory", "target", "universal", "stub-application.zip"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())

				Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "test-directory", "target", "universal", "stub-application.zip")))
			})

		})

		context("$BP_SBT_BUILT_ARTIFACT", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_SBT_BUILT_ARTIFACT", "test-directory/stub-application.zip")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_SBT_BUILT_ARTIFACT")).To(Succeed())
			})

			it("passes with BP_SBT_BUILT_ARTIFACT", func() {
				Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "test-directory"), 0755)).To(Succeed())

				in, err := os.Open(filepath.Join("testdata", "stub-application.zip"))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "test-directory", "stub-application.zip"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())

				Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "test-directory", "stub-application.zip")))
			})

		})
	})
}
