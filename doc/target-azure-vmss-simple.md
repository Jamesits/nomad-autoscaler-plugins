# target-azure-vmss-simple

Scales Azure virtual machine scale set. A fork of [the official Azure VMSS target plugin](https://developer.hashicorp.com/nomad/tools/autoscaling/plugins/target/azure-vmss).

Changes:

- Does not rely on a working Nomad cluster

## Configuration

### Agent Configuration

```hcl
target "azure-vmss" {
  driver = "target-azure-vmss-simple"
  config = {
    # required
    subscription_id   = ""
    
    # optional if using managed identities, required if running externally
    tenant_id         = ""
    client_id         = ""
    secret_access_key = ""
  }
}
```

### Policy Configuration

```hcl
scaling "example" {
  # ...

  policy {
    check "clients-azure-vmss" {
      target "azure-vmss" {
        resource_group = "resource-group-name"
        vm_scale_set   = "vmss-name"
      }

      strategy "example" {
        # ...
      }
    }

    target "example" {
        # ...
    }
  }
}
```
