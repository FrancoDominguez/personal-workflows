// Package function is the Cloud Functions entrypoint. The Go buildpack only
// imports the module root, so each function package is blank-imported here to
// run its init() registration; GOOGLE_FUNCTION_TARGET picks which one serves.
package function

import (
	_ "francodominguez.com/personal-workflows/app/functions/igfollowers" // registers fetchIgFollowers
	_ "francodominguez.com/personal-workflows/app/functions/unsub"       // registers autoEmailUnsubscribe
)
