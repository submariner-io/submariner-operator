/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/submariner-io/submariner-operator/pkg/internal/env"
	"github.com/submariner-io/submariner-operator/pkg/internal/log"
)

type Result int

const (
	Success Result = iota
	Failure
	Warning
)

// Status is used to track ongoing status in a CLI, with a nice loading spinner
// when attached to a terminal.
type Status struct {
	spinner *Spinner
	status  string
	logger  log.Logger
	// for controlling coloring etc.
	successFormat string
	failureFormat string
	warningFormat string
	// message queues
	successQueue []string
	failureQueue []string
	warningQueue []string
}

func NewStatus() *Status {
	var writer io.Writer = os.Stderr
	if env.IsSmartTerminal(writer) {
		writer = NewSpinner(writer)
	}

	return StatusForLogger(NewLogger(writer, 0))
}

// StatusForLogger returns a new status object for the logger l,
// if l is the kind cli logger and the writer is a Spinner, that spinner
// will be used for the status.
func StatusForLogger(l log.Logger) *Status {
	s := &Status{
		logger:        l,
		successFormat: " ✓ %s\n",
		failureFormat: " ✗ %s\n",
		warningFormat: " ⚠ %s\n",
		successQueue:  []string{},
		failureQueue:  []string{},
		warningQueue:  []string{},
	}
	// if we're using the CLI logger, check for if it has a spinner setup
	// and wire the status to that.
	if v, ok := l.(*Logger); ok {
		if v2, ok := v.writer.(*Spinner); ok {
			s.spinner = v2
			// use colored success / failure / warning messages.
			s.successFormat = " \x1b[32m✓\x1b[0m %s\n"
			s.failureFormat = " \x1b[31m✗\x1b[0m %s\n"
			s.warningFormat = " \x1b[33m⚠\x1b[0m %s\n"
		}
	}

	return s
}

// Start starts a new phase of the status, if attached to a terminal
// there will be a loading spinner with this status.
func (s *Status) Start(status string) {
	s.End(Success)
	s.status = status

	if s.spinner != nil {
		s.spinner.SetSuffix(fmt.Sprintf(" %s ", s.status))
		s.spinner.Start()
	} else {
		s.logger.V(0).Infof(" • %s  ...\n", s.status)
	}
}

// End completes the current status, ending any previous spinning and
// marking the status as success or failure.
func (s *Status) End(output Result) {
	if s.status == "" {
		return
	}

	if s.spinner != nil {
		s.spinner.Stop()
		fmt.Fprint(s.spinner.writer, "\r")
	}

	switch output {
	case Success:
		s.logger.V(0).Infof(s.successFormat, s.status)
	case Failure:
		s.logger.V(0).Infof(s.failureFormat, s.status)
	case Warning:
		s.logger.V(0).Infof(s.warningFormat, s.status)
	}

	for _, message := range s.successQueue {
		s.logger.V(0).Infof(s.successFormat, message)
	}

	for _, message := range s.failureQueue {
		s.logger.V(0).Infof(s.failureFormat, message)
	}

	for _, message := range s.warningQueue {
		s.logger.V(0).Infof(s.warningFormat, message)
	}

	s.status = ""
	s.successQueue = []string{}
	s.failureQueue = []string{}
	s.warningQueue = []string{}
}

func (s *Status) EndWithFailure(message string, a ...interface{}) {
	s.QueueFailureMessage(fmt.Sprintf(message, a...))
	s.End(Failure)
}

func (s *Status) EndWithSuccess(message string, a ...interface{}) {
	s.QueueSuccessMessage(fmt.Sprintf(message, a...))
	s.End(Success)
}

func (s *Status) EndWithWarning(message string, a ...interface{}) {
	s.QueueWarningMessage(fmt.Sprintf(message, a...))
	s.End(Warning)
}

// QueueSuccessMessage queues up a message, which will be displayed once
// the status ends (using the success format).
func (s *Status) QueueSuccessMessage(message string) {
	s.successQueue = append(s.successQueue, message)
}

// QueueFailureMessage queues up a message, which will be displayed once
// the status ends (using the failure format).
func (s *Status) QueueFailureMessage(message string) {
	s.failureQueue = append(s.failureQueue, message)
}

// QueuewarningMessage queues up a message, which will be displayed once
// the status ends (using the warning format).
func (s *Status) QueueWarningMessage(message string) {
	s.warningQueue = append(s.warningQueue, message)
}

func (s *Status) HasFailureMessages() bool {
	return len(s.failureQueue) > 0
}

func (s *Status) HasWarningMessages() bool {
	return len(s.warningQueue) > 0
}

func (s *Status) ResultFromMessages() Result {
	if s.HasFailureMessages() {
		return Failure
	}

	if s.HasWarningMessages() {
		return Warning
	}

	return Success
}

func CheckForError(err error) Result {
	if err == nil {
		return Success
	}

	return Failure
}
