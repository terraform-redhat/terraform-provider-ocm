package util

import (
	"github.com/openshift-online/ocm-sdk-go/logging"
	"os"
	"strings"
)

// The plugin infrastructure redirects the log package output so that it is sent to the main
// Terraform process, so if we want to have the logs of the SDK redirected we need to use
// the log package as well.
// no error can be returned
var Logger, _ = logging.NewGoLoggerBuilder().
	Error(true).
	Warn(true).
	Info(true).
	Debug(strings.EqualFold(os.Getenv("TF_LOG"), "DEBUG")).
	Build()
