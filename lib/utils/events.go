package utils

import (
	"time"

	"github.com/gravitational/reporting"
	log "github.com/sirupsen/logrus"
)

func RecordEvent(recorder reporting.Client, event reporting.Event) {
	if recorder == nil {
		log.Debug("event recorder not initialized")
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	recorder.Record(event)
}
