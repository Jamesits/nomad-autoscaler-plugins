package main

import (
	"github.com/Jamesits/nomad-autoscaler-plugins/pkg/plugins/strategy/pid"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
)

func main() {
	plugins.Serve(factory)
}

func factory(log hclog.Logger) interface{} {
	return pid.NewPIDPlugin(log)
}
