package utils

import (
	"fmt"
	"time"

	"github.com/gravitational/reporting"
	log "github.com/sirupsen/logrus"
)

func RecordEvent(recorder reporting.Client, event reporting.Event) {
	if recorder == nil {
		log.Debugf("event recorder not initialized, discarding event: %v", event)
		fmt.Println("discarding event:", event)
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	recorder.Record(event)
	fmt.Println("recorded event:", event)
}
