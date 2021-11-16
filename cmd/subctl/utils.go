package subctl

import (
	"fmt"
	"github.com/submariner-io/submariner-operator/pkg/version"
	"os"
)

// exitOnError will print your error nicely and exit in case of error
func exitOnError(message string, err error) {
	if err != nil {
		exitWithErrorMsg(fmt.Sprintf("%s: %s", message, err))
	}
}

// exitWithErrorMsg will print the message and quit the program with an error code
func exitWithErrorMsg(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "")
	version.PrintSubctlVersion(os.Stderr)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}
