package main

import (
	"github.com/Jamesits/nomad-autoscaler-plugins/pkg/plugins/apm/gitlab-ci"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
)

func main() {
	plugins.Serve(factory)
}

func factory(l hclog.Logger) interface{} {
	return gitlab_ci.NewGitLabPlugin(l)
}
