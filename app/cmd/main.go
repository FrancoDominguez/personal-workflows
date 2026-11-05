// This is the real production entrypoint: the buildpack compiles it because
// GOOGLE_BUILDABLE points here. Importing the root package registers every
// function; funcframework then serves the one named by FUNCTION_TARGET.
package main

import (
	"log"
	"os"

	_ "francodominguez.com/personal-workflows/app"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	hostname := ""
	if localOnly := os.Getenv("LOCAL_ONLY"); localOnly == "true" {
		hostname = "127.0.0.1"
	}

	log.Printf("Starting server on %s:%s", hostname, port)
	if err := funcframework.StartHostPort(hostname, port); err != nil {
		log.Fatalf("funcframework.StartHostPort: %v", err)
	}
}
