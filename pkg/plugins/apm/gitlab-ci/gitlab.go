package gitlab_ci

import (
	"context"
	"github.com/hasura/go-graphql-client"
	"time"
)

type CiJobStatus string

func (_ CiJobStatus) GetGraphQLType() string { return "CiJobStatus" }

type queryGetAllJobs struct {
	Jobs struct {
		Nodes []struct {
			Status     string    `graphql:"status"`
			CreatedAt  time.Time `graphql:"createdAt"`
			FinishedAt time.Time `graphql:"finishedAt"`
			Duration   uint      `graphql:"duration"`
			Active     bool      `graphql:"active"`
			Stuck      bool      `graphql:"stuck"`
			Tags       []string  `graphql:"tags"`
		} `graphql:"nodes"`
	} `graphql:"jobs(first: $first, statuses: $statuses)"`
}

func (n *APMPlugin) listJobs(ctx context.Context, limit int) (*queryGetAllJobs, error) {
	jobs := &queryGetAllJobs{}

	err := n.gqlClient.Query(ctx, jobs, map[string]interface{}{
		"first": &limit,
		// we need every jobs that runs or can be run
		"statuses": []CiJobStatus{"PREPARING", "PENDING", "RUNNING", "SUCCESS", "FAILED", "CANCELED"},
	}, graphql.OperationName("getAllJobs"))

	if err != nil {
		return nil, err
	}

	return jobs, nil
}
