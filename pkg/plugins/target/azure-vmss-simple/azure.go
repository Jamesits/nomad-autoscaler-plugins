// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure_vmss_simple

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/hashicorp/nomad-autoscaler/sdk/helper/ptr"
	"os"
)

// argsOrEnv allows you to pick an environmental variable for a setting if the arg is not set
func argsOrEnv(args map[string]string, key, env string) string {
	if value, ok := args[key]; ok {
		return value
	}
	return os.Getenv(env)
}

// setupAzureClients takes the passed config mapping and instantiates the
// required Azure service clients.
func (t *TargetPlugin) setupAzureClient(config map[string]string) error {
	var authorizer autorest.Authorizer
	// check for environmental variables, and use if the argument hasn't been set in config
	tenantID := argsOrEnv(config, configKeyTenantID, "ARM_TENANT_ID")
	clientID := argsOrEnv(config, configKeyClientID, "ARM_CLIENT_ID")
	subscriptionID := argsOrEnv(config, configKeySubscriptionID, "ARM_SUBSCRIPTION_ID")
	secretKey := argsOrEnv(config, configKeySecretKey, "ARM_CLIENT_SECRET")

	// Try to use the argument and environment provided arguments first, if this fails fall back to the Azure
	// SDK provided methods
	if tenantID != "" && clientID != "" && secretKey != "" {
		var err error
		authorizer, err = auth.NewClientCredentialsConfig(clientID, secretKey, tenantID).Authorizer()
		if err != nil {
			return fmt.Errorf("azure-vmss (ClientCredentials): %s", err)
		}
	} else {
		var err error
		authorizer, err = auth.NewAuthorizerFromEnvironment()
		if err != nil {
			return fmt.Errorf("azure-vmss (EnvironmentCredentials): %s", err)
		}
	}

	vmss := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	vmss.Sender = autorest.CreateSender()
	vmss.Authorizer = authorizer

	t.vmss = vmss

	return nil
}

// scale updates the Scale Set desired count to match what the
// Autoscaler has deemed required.
func (t *TargetPlugin) scale(ctx context.Context, resourceGroup string, vmScaleSet string, count int64, config map[string]string) error {

	// Create a logger for this action to pre-populate useful information we
	// would like on all log lines.
	log := t.logger.With("action", "scale", "vmss_name", vmScaleSet, "desired_count", count)

	future, err := t.vmss.Update(ctx, resourceGroup, vmScaleSet, compute.VirtualMachineScaleSetUpdate{
		Sku: &compute.Sku{
			Capacity: ptr.Of(count),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get the vmss update response: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, t.vmss.Client)
	if err != nil {
		return fmt.Errorf("cannot get the vmss update future response: %v", err)
	}

	log.Info("successfully performed and verified scaling out")
	return nil
}
