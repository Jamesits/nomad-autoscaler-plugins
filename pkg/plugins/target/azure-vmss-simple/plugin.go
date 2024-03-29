// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure_vmss_simple

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/plugins/target"
	"github.com/hashicorp/nomad-autoscaler/sdk"
)

const (
	// pluginName is the unique name of this plugin amongst Target plugins.
	pluginName = "target-azure-vmss-simple"

	// configKeys represents the known configuration parameters required at
	// varying points throughout the plugins lifecycle.
	configKeySubscriptionID = "subscription_id"
	configKeyTenantID       = "tenant_id"
	configKeyClientID       = "client_id"
	configKeySecretKey      = "secret_access_key"
	configKeyResoureGroup   = "resource_group"
	configKeyVMSS           = "vm_scale_set"
)

var (
	pluginInfo = &base.PluginInfo{
		Name:       pluginName,
		PluginType: sdk.PluginTypeTarget,
	}
)

// Assert that TargetPlugin meets the target.Target interface.
var _ target.Target = (*TargetPlugin)(nil)

// TargetPlugin is the Azure VMSS implementation of the target.Target interface.
type TargetPlugin struct {
	config map[string]string
	logger hclog.Logger
	vmss   compute.VirtualMachineScaleSetsClient
}

// NewAzureVMSSPlugin returns the Azure VMSS implementation of the target.Target
// interface.
func NewAzureVMSSPlugin(log hclog.Logger) *TargetPlugin {
	return &TargetPlugin{
		logger: log,
	}
}

// SetConfig satisfies the SetConfig function on the base.Base interface.
func (t *TargetPlugin) SetConfig(config map[string]string) error {

	t.config = config

	if err := t.setupAzureClient(config); err != nil {
		return err
	}

	return nil
}

// PluginInfo satisfies the PluginInfo function on the base.Base interface.
func (t *TargetPlugin) PluginInfo() (*base.PluginInfo, error) {
	return pluginInfo, nil
}

// Scale satisfies the Scale function on the target.Target interface.
func (t *TargetPlugin) Scale(action sdk.ScalingAction, config map[string]string) error {
	// Azure can't support dry-run like Nomad, so just exit.
	if action.Count == sdk.StrategyActionMetaValueDryRunCount {
		return nil
	}

	// We cannot scale a Scale Set without knowing the resource group and name.
	resourceGroup, ok := config[configKeyResoureGroup]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyResoureGroup)
	}
	vmScaleSet, ok := config[configKeyVMSS]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyVMSS)
	}
	ctx := context.Background()

	currVMSS, err := t.vmss.Get(ctx, resourceGroup, vmScaleSet)
	if err != nil {
		return fmt.Errorf("failed to get Azure vmss: %v", err)
	}

	capacity := *currVMSS.Sku.Capacity

	if capacity == action.Count {
		t.logger.Info("scaling not required", "resource_group", resourceGroup, "vmss", vmScaleSet,
			"current_count", capacity, "strategy_count", action.Count)
		return nil
	}

	err = t.scale(ctx, resourceGroup, vmScaleSet, action.Count, config)

	// If we received an error while scaling, format this with an outer message
	// so, it's nice for the operators and then return any error to the caller.
	if err != nil {
		err = fmt.Errorf("failed to perform scaling action: %v", err)
	}
	return err
}

// Status satisfies the Status function on the target.Target interface.
func (t *TargetPlugin) Status(config map[string]string) (*sdk.TargetStatus, error) {
	// We cannot scale a vmss without knowing the vmss resource group and name.
	resourceGroup, ok := config[configKeyResoureGroup]
	if !ok {
		return nil, fmt.Errorf("required config param %s not found", configKeyResoureGroup)
	}
	vmScaleSet, ok := config[configKeyVMSS]
	if !ok {
		return nil, fmt.Errorf("required config param %s not found", configKeyVMSS)
	}

	ctx := context.Background()

	vmss, err := t.vmss.Get(ctx, resourceGroup, vmScaleSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure ScaleSet: %v", err)
	}

	instanceView, err := t.vmss.GetInstanceView(ctx, resourceGroup, vmScaleSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure ScaleSet Instance View: %v", err)
	}

	// Set our initial status.
	resp := sdk.TargetStatus{
		Ready: true,
		Count: *vmss.Sku.Capacity,
		Meta:  make(map[string]string),
	}

	processInstanceView(instanceView, &resp)

	return &resp, nil
}

// processInstanceView updates the status object based on the details within
// the vmss instances.
func processInstanceView(instanceView compute.VirtualMachineScaleSetInstanceView, status *sdk.TargetStatus) {

	for _, instanceStatus := range *instanceView.VirtualMachine.StatusesSummary {
		if *instanceStatus.Code != "ProvisioningState/succeeded" {
			status.Ready = false
		}
	}

	latestTime := int64(math.MinInt64)
	for _, instanceStatus := range *instanceView.Statuses {
		if *instanceStatus.Code != "ProvisioningState/succeeded" {
			status.Ready = false
		}

		// Time isn't always populated, especially if the activity has not yet
		// finished :).
		if instanceStatus.Time != nil {
			currentTime := instanceStatus.Time.Time.UnixNano()
			if currentTime > latestTime {
				latestTime = currentTime
				status.Meta[sdk.TargetStatusMetaKeyLastEvent] = strconv.FormatInt(currentTime, 10)
			}
		}
	}
}
