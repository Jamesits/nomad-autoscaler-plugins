package gitlab_ci

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	url   = "https://gitlab.com/api/graphql"
	token = "glpat-***"
	tags  = ""
)

func TestQuery(t *testing.T) {
	plugin := NewGitLabPlugin(hclog.Default())

	// initialize
	err := plugin.SetConfig(map[string]string{
		"graphql_endpoint": url,
		"token":            token,
		"tags":             tags,
	})
	assert.NoError(t, err)

	now := time.Now()
	result, err := plugin.Query("", sdk.TimeRange{
		From: now.Add(-60 * time.Minute),
		To:   now,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result), 0)
}
