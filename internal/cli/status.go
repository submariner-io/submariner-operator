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
	"unicode"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/env"
	"github.com/submariner-io/submariner-operator/internal/log"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

type Result int

const (
	Success Result = iota
	Failure
	Warning
)

type (
	successType string
	warningType string
	failureType string
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
	// message queue
	messageQueue []interface{}
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
		messageQueue:  []interface{}{},
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

func NewReporter() reporter.Interface {
	return NewStatus()
}

// Start starts a new phase of the status, if attached to a terminal
// there will be a loading spinner with this status.
func (s *Status) Start(message string, args ...interface{}) {
	s.End()
	s.status = fmt.Sprintf(message, args...)

	if s.spinner != nil {
		s.spinner.SetSuffix(fmt.Sprintf(" %s ", s.status))
		s.spinner.Start()
	} else {
		s.logger.V(0).Infof(" • %s  ...\n", s.status)
	}
}

// EndWith completes the current status, ending any previous spinning and
// marking the status as success or failure.
func (s *Status) EndWith(output Result) {
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

	for _, message := range s.messageQueue {
		switch m := message.(type) {
		case successType:
			s.logger.V(0).Infof(s.successFormat, m)
		case failureType:
			s.logger.V(0).Infof(s.failureFormat, m)
		case warningType:
			s.logger.V(0).Infof(s.warningFormat, m)
		}
	}

	s.status = ""
	s.messageQueue = []interface{}{}
}

func (s *Status) EndWithFailure(message string, a ...interface{}) {
	s.QueueFailureMessage(fmt.Sprintf(message, a...))
	s.EndWith(Failure)
}

func (s *Status) EndWithSuccess(message string, a ...interface{}) {
	s.QueueSuccessMessage(fmt.Sprintf(message, a...))
	s.EndWith(Success)
}

func (s *Status) EndWithWarning(message string, a ...interface{}) {
	s.QueueWarningMessage(fmt.Sprintf(message, a...))
	s.EndWith(Warning)
}

// QueueSuccessMessage queues up a message, which will be displayed once
// the status ends (using the success format).
func (s *Status) QueueSuccessMessage(message string) {
	s.messageQueue = append(s.messageQueue, successType(message))
}

// QueueFailureMessage queues up a message, which will be displayed once
// the status ends (using the failure format).
func (s *Status) QueueFailureMessage(message string) {
	s.messageQueue = append(s.messageQueue, failureType(message))
}

// QueueWarningMessage queues up a message, which will be displayed once
// the status ends (using the warning format).
func (s *Status) QueueWarningMessage(message string) {
	s.messageQueue = append(s.messageQueue, warningType(message))
}

func (s *Status) HasFailureMessages() bool {
	for _, message := range s.messageQueue {
		if _, ok := message.(failureType); ok {
			return true
		}
	}

	return false
}

func (s *Status) HasWarningMessages() bool {
	for _, message := range s.messageQueue {
		if _, ok := message.(warningType); ok {
			return true
		}
	}

	return false
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

// Failure queues up a message, which will be displayed once
// the status ends (using the failure format).
func (s *Status) Failure(message string, a ...interface{}) {
	if message == "" {
		return
	}

	if s.status != "" {
		s.messageQueue = append(s.messageQueue, failureType(fmt.Sprintf(message, a...)))
	} else {
		s.logger.V(0).Infof(s.failureFormat, fmt.Sprintf(message, a...))
	}
}

// Success queues up a message, which will be displayed once
// the status ends (using the warning format).
func (s *Status) Success(message string, a ...interface{}) {
	if message == "" {
		return
	}

	if s.status != "" {
		s.messageQueue = append(s.messageQueue, successType(fmt.Sprintf(message, a...)))
	} else {
		s.logger.V(0).Infof(s.successFormat, fmt.Sprintf(message, a...))
	}
}

// Warning queues up a message, which will be displayed once
// the status ends (using the warning format).
func (s *Status) Warning(message string, a ...interface{}) {
	if message == "" {
		return
	}

	if s.status != "" {
		s.messageQueue = append(s.messageQueue, warningType(fmt.Sprintf(message, a...)))
	} else {
		s.logger.V(0).Infof(s.warningFormat, fmt.Sprintf(message, a...))
	}
}

func (s *Status) Error(err error, message string, args ...interface{}) error {
	err = errors.Wrapf(err, message, args...)
	if err == nil {
		return nil
	}

	capitalizeFirst := func(str string) string {
		for i, v := range str {
			return string(unicode.ToUpper(v)) + str[i+1:]
		}

		return ""
	}

	s.Failure(capitalizeFirst(err.Error()))
	s.End()

	return err
}

// End completes the current status, ending any previous spinning and
// marking the status as success or failure.
func (s *Status) End() {
	s.EndWith(s.ResultFromMessages())
}
