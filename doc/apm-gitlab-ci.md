# apm-gitlab-ci

Nomad Autoscaler plugin for reading GitLab CI running/pending job count.

## Configuration

### Agent Configuration

```hcl
apm "gitlab-ci" {
  driver = "apm-gitlab-ci"
  args   = [] # no args supported

  config = {
    # Required: your GitLab instance's GraphQL API endpoint
    graphql_endpoint = "https://gitlab.example.com/api/graphql"
    # Required: an access token, must have admin access
    token = "glpat-***"
    
    # optional: how much jobs to query
    query_job_limit = "200",
    # optional: reports time series data in which frequency back to nomad-autoscaler
	sample_interval_secs = "60",
	# optional: runner tag filter
	tags = "",
  }
}
```

Tags format: a comma separated list of tags, where excluded tags have a `-` in the front. Spaces around the tag does not matter. e.g. `tag_a, tag_b, -exclusion_tag_c, ...`

Matching rules:
- Any jobs matching any single exclusion tag is excluded
- If any inclusion tags are set, then a job not matching any one of the inclusion tags is excluded

### Policy Configuration

```hcl
scaling "example" {
  # ...

  policy {
    check "job_count" {
      source = "gitlab-ci"
      query  = "tags:\"linux\""

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

The query string should be of [Go StructTag convention format](https://pkg.go.dev/reflect#StructTag). Supported keys:

- `tags`: see "Tags format" above
