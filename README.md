# `gcr.io/paketo-buildpacks/sbt`

The Paketo SBT Buildpack is a Cloud Native Buildpack that builds SBT-based applications from source.

## Behavior

This buildpack will participate all the following conditions are met

* `<APPLICATION_ROOT>/build.sbt` exists

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.sbt` to a layer for caching
* If `<APPLICATION_ROOT>/sbt` exists
  * Runs `<APPLICATION_ROOT>/sbt universal:packageBin` to build the application
* If `<APPLICATION_ROOT>/sbt` does not exist
  * Contributes SBT to a layer with all commands on `$PATH`
  * Runs `<SBT_ROOT>/bin/sbt package` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* If `$BP_SBT_BUILT_ARTIFACT` matched a single file
  * Restores `$BP_SBT_BUILT_ARTIFACT` from the layer, expands the single file to `<APPLICATION_ROOT>`
* If `$BP_SBT_BUILT_ARTIFACT` matched a directory or multiple files
  * Restores the files matched by `$BP_SBT_BUILT_ARTIFACT` to `<APPLICATION_ROOT>`

## Configuration

| Environment Variable      | Description                                                                                                                                                                                                                        |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$BP_SBT_BUILD_ARGUMENTS` | Configure the arguments to pass to build system. Defaults to `universal:packageBin`.                                                                                                                                               |
| `$BP_SBT_BUILT_MODULE`    | Configure the module to find application artifact in. Defaults to the root module (empty).                                                                                                                                         |
| `$BP_SBT_BUILT_ARTIFACT`  | Configure the built application artifact explicitly. Supersedes `$BP_SBT_BUILT_MODULE`. Defaults to `target/universal/*.zip`. Can match a single file, multiple files or a directory. Can be one or more space separated patterns. |

## Bindings

The buildpack optionally accepts the following bindings:

### Type: `dependency-mapping`

| Key                   | Value   | Description                                                                                       |
| --------------------- | ------- | ------------------------------------------------------------------------------------------------- |
| `<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>` |

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

