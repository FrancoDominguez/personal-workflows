// Package function is the Cloud Functions entrypoint. The Go buildpack only
// imports the module root, so the functions package is blank-imported here to
// run its init() registrations; GOOGLE_FUNCTION_TARGET picks which one serves.
package function

import (
	_ "francodominguez.com/gmail-webhook/app/functions"
)
