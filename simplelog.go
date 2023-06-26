// Package simplelog is a logging package with the focus on simplicity and
// ease of use. It utilizes the log package from the standard library with
// some advanced features.
// Once started, the simple logger runs as a service and listens for logging
// requests through the functions WriteTo[Stdout|File|Multiple].
// As the name of the WriteTo functions suggests, the simple logger writes
// to either standard out, a log file, or multiple targets.
package simplelog

// message catalog
const (
	m001 = "log file not initialized"
	m002 = "log service was already started"
	m003 = "log service is not running"
	m004 = "log service has not been started"
)

// Startup starts the log service.
// The bufferSize specifies the number of log messages which can be buffered before the log service blocks.
// The log service runs in its own goroutine.
func Startup(bufferSize int) {
	if !c.checkState(running) {
		// start the log service
		s.setAttribut(logbuffer, 10)
		c.service(start)
	} else {
		panic(m002)
	}
}

// Shutdown stops the log service and does some cleanup.
// Before the log service is stopped, all pending log messages are flushed and resources are released.
func Shutdown() {
	if c.checkState(running) {
		// stop the log service
		c.service(stop)
	} else {
		panic(m003)
	}
}

// InitLogFile initializes the log file.
func InitLogFile(logName string) {
	if c.checkState(running) {
		// initialize the log file
		s.config <- configMessage{initlog, logName}
		<-s.confirmed
	} else {
		panic(m004)
	}
}

// ChangeLogName changes the log file name.
// As part of this task, the current log file is closed (not deleted) and a log file with the new name is created.
// The log service doesn't need to be stopped for this task.
func ChangeLogName(newLogName string) {
	if c.checkState(running) {
		// change the log name
		s.config <- configMessage{changelog, newLogName}
		<-s.confirmed
	} else {
		panic(m004)
	}
}

// WriteToStdout writes a log message to stdout.
func WriteToStdout(values ...any) {
	if c.checkState(running) {
		msg := parseValues(values)
		s.data <- logMessage{stdout, msg}
	} else {
		panic(m004)
	}
}

// WriteToFile writes a log message to a log file.
func WriteToFile(values ...any) {
	if c.checkState(running) {
		if s.fileDesc == nil {
			panic(m001)
		}
		msg := parseValues(values)
		s.data <- logMessage{file, msg}
	} else {
		panic(m004)
	}
}

// WriteToMulti writes a log message to multiple targets.
func WriteToMulti(values ...any) {
	if c.checkState(running) {
		if s.fileDesc == nil {
			panic(m001)
		}
		msg := parseValues(values)
		s.data <- logMessage{multi, msg}
	} else {
		panic(m004)
	}
}
