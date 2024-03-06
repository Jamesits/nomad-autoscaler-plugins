# nomad-autoscaler-plugins

A random collection of [Hashicorp Nomad Autoscaler](https://github.com/hashicorp/nomad-autoscaler) plugins I use.

## Plugins

| ID                       | Type     | Status | Usage                                                                                | Documentation                          |
|--------------------------|----------|--------|--------------------------------------------------------------------------------------|----------------------------------------|
| apm-gitlab-ci            | APM      | Works  | Reads GitLab CI running/pending job count                                            | [doc](doc/apm-gitlab-ci.md)            |
| strategy-pid             | Strategy | Works  | Proportional–integral–derivative controller algorithm                                | [doc](doc/strategy-pid.md)             |
| target-azure-vmss-simple | Target   | Works  | Scales Azure virtual machine scale set, but does not require a working Nomad cluster | [doc](doc/target-azure-vmss-simple.md) |

Notes:
- If you use the Linux packages generated by GoReleaser, set plugin dir to `/usr/lib/nomad-autoscaler/plugins`.

## Development

Requirements: `go upx goreleaser`

### Local

```shell
# build plugins
./build_local.sh
# launch nomad-autoscaler
nomad-autoscaler agent -plugin-dir=./dist/plugins #...other args...#
```

### Release

```shell
goreleaser release --clean
```
