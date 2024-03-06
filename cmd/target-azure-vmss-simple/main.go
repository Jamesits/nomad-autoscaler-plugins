// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	azure "github.com/Jamesits/nomad-autoscaler-plugins/pkg/plugins/target/azure-vmss-simple"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
)

func main() {
	plugins.Serve(factory)
}

// factory returns a new instance of the Azure VMSS plugin.
func factory(log hclog.Logger) interface{} {
	return azure.NewAzureVMSSPlugin(log)
}
