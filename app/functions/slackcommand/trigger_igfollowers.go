package slackcommand

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	scheduler "cloud.google.com/go/scheduler/apiv1"
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
)

// triggerIgFollowers forces the `igfollowers-weekly` Cloud Scheduler job to run
// now instead of calling the igfollowers service directly.
//
// Why not call the service? A follower scrape takes ~30s and Slack drops any
// slash command that hasn't answered within 3s. Cloud Run would also cancel
// the scrape the moment we hung up on it. Cloud Scheduler's RunJob returns as
// soon as the run is queued; the job then invokes igfollowers with its own
// OIDC token and deadline, exactly like the Monday schedule does, and
// igfollowers posts its own result to Slack (only on unfollows or failure).
//
// The runtime service account needs roles/cloudscheduler.jobRunner (granted
// project-wide in terraform/service_slack_command.tf: Cloud Scheduler has no
// per-job IAM).
func triggerIgFollowers(ctx context.Context) (string, error) {
	job := os.Getenv("IGFOLLOWERS_JOB")
	if job == "" {
		return "", errors.New("IGFOLLOWERS_JOB is not set")
	}

	// Leave headroom under Slack's 3s limit for the reply itself.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	client, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return "", fmt.Errorf("creating scheduler client: %w", err)
	}
	defer client.Close()

	if _, err := client.RunJob(ctx, &schedulerpb.RunJobRequest{Name: job}); err != nil {
		return "", fmt.Errorf("running job %s: %w", job, err)
	}

	return "Started the IG followers check. You'll get a message only if someone unfollowed you or the job fails (usually within a minute).", nil
}
