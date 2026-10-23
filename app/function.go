package function

import (
	"log"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("scanEmail", scanEmail)
}

func scanEmail(w http.ResponseWriter, r *http.Request) {
	log.Printf("Scanning email!")
	fmt.Fprintln(w, "Scanning email!")
}









