package simplelog

import (
	"log"
	"os"
)

// log targets
const (
	stdout = 1 << iota     // write the log record to STDOUT
	file                   // write the log record to the log file
	multi  = stdout | file // write the log record to STDOUT and to the log file
)

// configuration categories
const (
	initlog   = iota // initializes a log file
	changelog        // change the log file name
)

// signal to confirm actions across channels
type signal struct{}

// A logMessage represents the log message which will be sent to the log service.
type logMessage struct {
	target int    // the log target bits, e.g. stdout, file, and so on.
	data   string // the payload of the log message, which will be sent to the log target
}

// A configMessage represents the config message which will be sent to the log service.
type configMessage struct {
	category int    // the configuration category bits, which are used to trigger certain config tasks by the log service, e.g. setlogname, changelogname, and so on.
	data     string // the data, which will be processed by a config task
}

// service is a configuration to handle simple log requests.
type service struct {
	data   chan logMessage    // the channel for sending log messages to the log service; this channel will be a buffered channel
	config chan configMessage // the channel for sending config messages to the log service
	stop   chan signal        // the channel for sending a stop signal to the log service
	done   chan signal        // the channel for sending a done signal to the caller

	sim simpleLog // the simple logger properties
}

type simpleLog struct {
	fileHandle *os.File            // the file handle of the log file
	logHandle  map[int]*log.Logger // a map which stores for every log target its assigned log handle
}

// global service instance
var s = &service{}

// isActive checks whether the log service is active.
func (s *service) isActive() bool {
	if s.data != nil {
		return true
	} else {
		return false
	}
}

// instance returns log handler instances for a given log target.
func (s *simpleLog) instance(target int) (*log.Logger, *log.Logger) {
	var log1, log2 *log.Logger
	switch target {
	case stdout:
		log1 = s.createsimpleLog(stdout)
	case file:
		log1 = s.createsimpleLog(file)
	case multi:
		// stdout and file log handler have different properties, thus io.MultiWriter can't be used
		log1 = s.createsimpleLog(stdout)
		log2 = s.createsimpleLog(file)
	}
	return log1, log2
}

// createsimpleLog checks if a simple logger exists for a specific target. If not, it will be created accordingly.
// Each log target is assinged its own log handler.
func (s *simpleLog) createsimpleLog(target int) *log.Logger {
	if _, found := s.logHandle[target]; !found {
		// log handler doesn't exists - create it
		switch target {
		case stdout:
			s.logHandle[stdout] = log.New(os.Stdout, "", 0)
		case file:
			s.logHandle[file] = log.New(s.fileHandle, "", log.Ldate|log.Ltime|log.Lmicroseconds)
			// the first 'file' log event always adds an empty line to the log file
			s.fileHandle.WriteString("\n")
		}
	}
	return s.logHandle[target]
}

// setupLogFile creates and opens the log file.
func (s *simpleLog) setupLogFile(logName string) {
	var err error
	s.fileHandle, err = os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
}

// changeLogFileName changes the name of the log file.
func (s *simpleLog) changeLogFileName(newLogName string) {
	// remove all file log handles from the logHandler map which are linked to the old log name
	delete(s.logHandle, file)
	s.fileHandle.Close()
	s.setupLogFile(newLogName)
}

// service represents the log service.
// This service function runs in a dedicated goroutine and will be started as part of the log service startup process.
// It listenes on the following channels:
//   - data
//   - config
//   - stop
func (s *service) service(alive chan<- signal) {
	var logMsg logMessage
	var cfgMsg configMessage

	// service is alive - send acknowledgment to caller
	alive <- signal{}

	// service loop
	for {
		select {
		case <-s.stop:
			// write all messages which are still in the data channel and have not been written yet
			s.flushMessages(len(s.data))
			s.done <- signal{}
			return
		case logMsg = <-s.data:
			s.writeMessage(logMsg)
		case cfgMsg = <-s.config:
			switch cfgMsg.category {
			case initlog:
				s.sim.setupLogFile(cfgMsg.data)
				s.done <- signal{}
			case changelog:
				// write all messages to the old log file, which were already sent to the data channel before the change log name was triggered
				s.flushMessages(len(s.data))
				// change the log file name
				s.sim.changeLogFileName(cfgMsg.data)
				s.done <- signal{}
			}
		}
	}
}

// writeMessage writes data of log messages to a dedicated target.
func (s *service) writeMessage(logMsg logMessage) {
	switch logMsg.target {
	case stdout:
		stdoutLogHandle, _ := s.sim.instance(stdout)
		stdoutLogHandle.Print(logMsg.data)
	case file:
		fileLogHandle, _ := s.sim.instance(file)
		fileLogHandle.Print(logMsg.data)
	case multi:
		stdoutLogHandle, fileLogHandle := s.sim.instance(multi)
		stdoutLogHandle.Print(logMsg.data)
		fileLogHandle.Print(logMsg.data)
	}
}

// flushMessages flushes(writes) a number of messages to a dedicated target.
// The messages will be read from a buffered channel.
// Buffered channels in Go are always FIFO, so messages are flushed in FIFO approach.
func (s *service) flushMessages(numMessages int) {
	for numMessages > 0 {
		s.writeMessage(<-s.data)
		numMessages--
	}
}