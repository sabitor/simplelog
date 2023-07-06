package simplelog

import (
	"bufio"
	"log"
	"os"
	"time"
)

// service instance
var s = new(logService)

// log targets
const (
	stdout = iota // write the log record to stdout
	file          // write the log record to the log file
	multi         // write the log record to stdout and to the log file
)

// log service actions
const (
	start = iota
	stop
	initlog
	newlog
)

// log service states bitmask
const (
	stopped = 1 << iota // the service is stopped and cannot process log requests
	running             // the service is running
)

// log service attributes
const (
	logbuffer = iota // defines the buffer size of the logMessage channel
	logfilename
)

// signal to confirm actions across channels
type signal struct{}

// a logMessage represents the log message which will be sent to the log service.
type logMessage struct {
	target int    // the log target bits, e.g. stdout, file, and so on.
	data   string // the payload of the log message, which will be sent to the log target
}

// a configMessage represents the config message which will be sent to the log service.
type configMessage struct {
	action int    // the configuration action, which is used to trigger certain config tasks by the log service
	data   string // the data, which will be used by the config task
}

// logService is structure used to handle workflows triggered by the simplelog API.
type logService struct {
	logFactory

	serviceConfig chan configMessage // the channel for sending config messages to the log service
	serviceStop   chan signal        // the channel for sending a stop signal to the log service
}

// logFactory is the base data collection to support logging to multiple targets.
type logFactory struct {
	attribute map[int]any     // the map which contains the log factory attributes
	logData   chan logMessage // the channel for sending log messages to the log service; this channel buffered
	multiLog                  // the multiLog supports logging to stdout and file
}

// stdoutLogWriter is a data collection to support logging to stdout.
type stdoutLog struct {
	stdoutLogInstance *log.Logger
}

// fileLogWriter is a data collection to support logging to files.
type fileLog struct {
	fileWriter      *bufio.Writer
	fileDesc        *os.File
	fileLogInstance *log.Logger
}

// logWriter is the log writer which supports logging to stdout and to files.
type multiLog struct {
	stdoutLog
	fileLog
}

// logWriter interface includes definitions of the following method signatures:
//   - instance
type logWriter interface {
	instance() *log.Logger // create and return a log.logger instance
}

// instance denotes the logWriter interface implementation by the stdoutLog type.
func (s *stdoutLog) instance() *log.Logger {
	if s.stdoutLogInstance == nil {
		s.stdoutLogInstance = log.New(os.Stdout, "", 0)
	}
	return s.stdoutLogInstance
}

// instance denotes the logWriter interface implementation by the fileLog type.
func (s *fileLog) instance() *log.Logger {
	if s.fileLogInstance == nil {
		if s.fileDesc == nil {
			panic(m001)
		}
		// s.fileWriter = bufio.NewWriter(s.fileDesc)
		s.fileWriter = bufio.NewWriterSize(s.fileDesc, 16384)
		// fmt.Println("Buffer size:", w.Size())
		// s.fileWriter = s.fileDesc
		s.fileLogInstance = log.New(s.fileWriter, "", log.Ldate|log.Ltime|log.Lmicroseconds)
		s.fileWriter.WriteString("\n")
		go func() {
			for {
				time.Sleep(2 * time.Second)
				if s.fileWriter.Buffered() > 0 {
					s.fileWriter.Flush()
				}
			}
		}()
	}
	return s.fileLogInstance
}

// getLogWriter returns a log.Logger instance.
func (s *multiLog) getLogWriter(lw logWriter) *log.Logger {
	return lw.instance()
}

// setupLogFile creates and opens the log file.
func (s *multiLog) setupLogFile(logName string) {
	var err error
	s.fileDesc, err = os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
}

func (s *multiLog) closeLogFile() {
	if s.fileDesc != nil {
		if s.fileWriter.Buffered() > 0 {
			s.fileWriter.Flush()
		}
		if err := s.fileDesc.Close(); err != nil {
			panic(err)
		}
		s.fileDesc = nil
	}
}

// changeLogFileName changes the name of the log file.
func (s *multiLog) changeLogFileName(newLogName string) {
	// close old log file
	s.closeLogFile()
	// close file log instance (link to old log descriptor still exists)
	s.fileLogInstance = nil
	s.setupLogFile(newLogName)
}

// setAttribut sets a log service attribute.
func (s *logService) setAttribut(key int, value any) {
	if s.attribute == nil {
		s.attribute = make(map[int]any)
	}
	s.attribute[key] = value
}

// run represents the log service.
// This service function runs in a dedicated goroutine and is started as part of the log service startup process.
// It handles client requests by listening on the following channels:
//   - stop
//   - data
//   - config
func (s *logService) run() {
	var logMsg logMessage
	var cfgMsg configMessage

	c.setState(running)
	defer c.setState(stopped)

	for {
		select {
		case <-s.serviceStop:
			s.flush()
			return
		case logMsg = <-s.logData:
			s.writeMessage(logMsg)
		case cfgMsg = <-s.serviceConfig:
			switch cfgMsg.action {
			case initlog:
				s.setupLogFile(cfgMsg.data)
				c.execServiceActionResponse <- signal{}
			case newlog:
				s.flush()
				s.changeLogFileName(cfgMsg.data)
				c.execServiceActionResponse <- signal{}
			}
		}
	}
}

// writeMessage writes data of log messages to a dedicated target.
func (s *logService) writeMessage(logMsg logMessage) {
	switch logMsg.target {
	case stdout:
		s.stdoutLog.instance().Print(logMsg.data)
	case file:
		s.fileLog.instance().Print(logMsg.data)
	case multi:
		s.stdoutLog.instance().Print(logMsg.data)
		s.fileLog.instance().Print(logMsg.data)
	}
}

// flush flushes(writes) messages, which are still buffered in the data channel.
// Buffered channels in Go are always FIFO, so messages are flushed in FIFO approach.
func (s *logService) flush() {
	for len(s.logData) > 0 {
		s.writeMessage(<-s.logData)
	}
}
