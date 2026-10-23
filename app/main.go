// Package app logs a line whenever Gmail reports inbox activity.
//
// The Pub/Sub message only carries a historyId cursor, not the email itself;
// fetching actual content via the Gmail API is a later brick.
package app

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
)

func init() {
	functions.CloudEvent("handle", handle)
}

func handle(ctx context.Context, e event.Event) error {
	log.Println("you got a new email")
	return nil
}
