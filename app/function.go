// Package function is the Cloud Functions entrypoint. The Go buildpack only
// looks at the module root, so each utility is blank-imported here to run its
// init() and register itself; GOOGLE_FUNCTION_TARGET picks which one serves.
package function

import (
	_ "francodominguez.com/gmail-webhook/app/internal/unsub" // registers autoEmailUnsubscribe
)
