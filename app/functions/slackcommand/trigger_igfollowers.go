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
