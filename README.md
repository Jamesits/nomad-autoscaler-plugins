# nomad-autoscaler-plugins

A random collection of [Hashicorp Nomad Autoscaler](https://github.com/hashicorp/nomad-autoscaler) plugins I use.

## Plugins

| ID                       | Type     | Status    | Usage                                                                                | Documentation                          |
|--------------------------|----------|-----------|--------------------------------------------------------------------------------------|----------------------------------------|
| apm-gitlab-ci            | APM      | It works! | Reads GitLab CI running/pending job count                                            | [doc](doc/apm-gitlab-ci.md)            |
| strategy-pid             | Strategy | WIP       | Proportional–integral–derivative controller algorithm                                | [doc](doc/strategy-pid.md)             |
| target-azure-vmss-simple | Target   | WIP       | Scales Azure virtual machine scale set, but does not require a working Nomad cluster | [doc](doc/target-azure-vmss-simple.md) |

## Development

Requirements: `go upx goreleaser`

Building locally: `./build_local.sh`

Launching: `nomad-autoscaler agent -plugin-dir=./dist/plugins #...other args...#`
