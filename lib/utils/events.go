package utils

import (
	"time"

	"github.com/gravitational/reporting"
)

func RecordEvent(recorder reporting.Client, event reporting.Event) {
	if recorder == nil {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	recorder.Record(event)
}
