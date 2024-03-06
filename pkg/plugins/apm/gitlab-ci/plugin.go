package gitlab_ci

import (
	"context"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins/apm"
	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/sdk"
	"github.com/hasura/go-graphql-client"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	pluginName = "apm-gitlab-ci"
)

var (
	pluginInfo = &base.PluginInfo{
		Name:       pluginName,
		PluginType: sdk.PluginTypeAPM,
	}

	defaultConfig = map[string]string{
		"graphql_endpoint":     "https://gitlab.com/api/graphql",
		"token":                "",
		"query_job_limit":      "200",
		"sample_interval_secs": "60",
		"tags":                 "",
	}
)

// Test interface compatibility
var _ apm.APM = (*APMPlugin)(nil)

type APMPlugin struct {
	logger hclog.Logger

	config         map[string]string
	queryJobLimit  int
	sampleInterval time.Duration
	includeTags    []string
	excludeTags    []string

	gqlClient *graphql.Client
}

func NewGitLabPlugin(log hclog.Logger) apm.APM {
	return &APMPlugin{
		logger: log,
	}
}

func (n *APMPlugin) PluginInfo() (*base.PluginInfo, error) {
	n.logger.Debug("PluginInfo() called")
	return pluginInfo, nil
}

func (n *APMPlugin) SetConfig(config map[string]string) error {
	n.logger.Debug("SetConfig() called", "config", config)

	n.config = make(map[string]string)
	// copy from default config
	for k, v := range defaultConfig {
		n.config[k] = v
	}
	// apply environment variables
	if os.Getenv("GITLAB_GRAPHQL_ENDPOINT") != "" {
		n.config["graphql_endpoint"] = os.Getenv("GITLAB_GRAPHQL_ENDPOINT")
	}
	if os.Getenv("GITLAB_TOKEN") != "" {
		n.config["token"] = os.Getenv("GITLAB_TOKEN")
	}
	// copy from user config
	for k, v := range config {
		n.config[k] = v
	}

	// debug print parsed config
	for k, v := range n.config {
		if k == "token" {
			n.logger.Trace("config item received", "token", strings.Repeat("*", len(v)))
			continue
		}

		n.logger.Trace("config item received", k, v)
	}

	l, err := strconv.ParseInt(n.config["query_job_limit"], 10, 64)
	if err != nil || l <= 0 {
		return fmt.Errorf("query_job_limit must be an positive integer, got %s instead: %w", n.config["query_job_limit"], err)
	}
	n.queryJobLimit = int(l)

	l, err = strconv.ParseInt(n.config["sample_interval_secs"], 10, 64)
	if err != nil || l <= 0 {
		return fmt.Errorf("sample_interval_secs must be an positive integer, got %s instead: %w", n.config["sample_interval_secs"], err)
	}
	n.sampleInterval = time.Second * time.Duration(l)

	i, e := splitTags(n.config["tags"])
	n.includeTags = append(n.includeTags, i...)
	n.excludeTags = append(n.excludeTags, e...)

	// TODO: custom HTTP timeout
	n.gqlClient = graphql.NewClient(n.config["graphql_endpoint"], &http.Client{
		Transport: &authenticatedHttpTransport{authHeader: "Bearer " + n.config["token"]},
	})
	return nil
}

func (n *APMPlugin) QueryMultiple(q string, r sdk.TimeRange) ([]sdk.TimestampedMetrics, error) {
	n.logger.Debug("QueryMultiple() called", "q", q, "range", r)
	m, err := n.Query(q, r)
	if err != nil {
		return nil, err
	}
	return []sdk.TimestampedMetrics{m}, nil
}

func (n *APMPlugin) Query(q string, r sdk.TimeRange) (sdk.TimestampedMetrics, error) {
	n.logger.Debug("Query() called", "query", q, "range", r)
	var result sdk.TimestampedMetrics

	// parse the query
	queryConfig, err := structtag.Parse(q)
	if err != nil {
		n.logger.Error("parse query failed: %v", err)
		return nil, err
	}
	tags, err := queryConfig.Get("tags")
	tagsValue := ""
	if tags != nil {
		tagsValue = tags.Value()
	}
	i, e := splitTags(tagsValue)
	includeTags := append(n.includeTags, i...)
	excludeTags := append(n.excludeTags, e...)
	n.logger.Trace("tags", "include", includeTags, "exclude", excludeTags)

	// list jobs
	ret, err := n.listJobs(context.Background(), 50)
	if err != nil {
		n.logger.Error("listJobs failed: %v", err)
		return nil, err
	}

	// parse jobs into a time series
	for now := r.From; now.Compare(r.To) <= 0; now = now.Add(n.sampleInterval) {
		readyJobs := 0
		runningJobs := 0
		pendingJobs := 0
		for _, j := range ret.Jobs.Nodes {
			// tag filter
			if ((len(includeTags) > 0) && !matchAny(j.Tags, includeTags)) || matchAny(j.Tags, excludeTags) {
				continue
			}

			// state replay
			// pending
			if (j.FinishedAt == time.Time{}) {
				pendingJobs++
				continue
			}
			startedAt := j.FinishedAt.Add(-time.Second * time.Duration(j.Duration))
			// running: (startedAt <= now < finishedAt)
			if (now.Compare(startedAt) >= 0) && (now.Compare(j.FinishedAt) < 0) {
				runningJobs++
				continue
			}
			// ready: (createdAt <= now < startedAt)
			if (now.Compare(j.CreatedAt) >= 0) && (now.Compare(startedAt) < 0) {
				readyJobs++
				continue
			}
		}

		n.logger.Trace("time series point", "time", now, "runningJobs", runningJobs, "readyJobs", readyJobs, "pendingJobs", pendingJobs)
		result = append(result, sdk.TimestampedMetric{
			Timestamp: now,
			Value:     float64(readyJobs + runningJobs + pendingJobs),
		})
	}

	n.logger.Debug("Query() returning", "last_result", result[len(result)-1].Value)
	n.logger.Trace("Query() returning", "result", result)
	return result, nil
}
