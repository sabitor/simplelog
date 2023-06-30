package simplelog

import (
	"strconv"
)

// control instance
var c = new(control)

// control is a structure used to control the log service and log service workflows.
type control struct {
	checkServiceState         chan int    // the channel for receiving a state check request from the caller
	checkServiceStateResponse chan bool   // the channel for sending a boolean response to the caller
	setServiceState           chan int    // the channel for receiving a state change request from the caller
	execServiceAction         chan int    // the channel for receiving a service action request from the caller
	execServiceActionResponse chan signal // the channel for sending a signal response to the caller to continue the workflow
}

// init starts the control.
// The control monitors the log service and triggers actions to be started by the log service.
func init() {
	c.checkServiceState = make(chan int)
	c.checkServiceStateResponse = make(chan bool)
	c.setServiceState = make(chan int)
	c.execServiceAction = make(chan int)
	c.execServiceActionResponse = make(chan signal)

	controlRunning := make(chan bool)
	go c.run(controlRunning)
	if !<-controlRunning {
		panic(m000)
	}
}

// run represents the control service.
// This utility function runs in a dedicated goroutine and is started when the init function is implicitly called.
// It handles requests by listening on the following channels:
//   - execServiceAction
//   - setServiceState
//   - checkServiceState
func (c *control) run(controlRunning chan bool) {
	var singleState, totalState int

	for {
		select {
		case controlRunning <- true:
		case action := <-c.execServiceAction:
			switch action {
			case start:
				// allocate log service channels
				buf, _ := strconv.Atoi(convertToString(s.attribute[logbuffer]))
				s.logData = make(chan logMessage, buf)
				s.serviceConfig = make(chan configMessage)
				s.serviceStop = make(chan signal, 1) // has to be buffered to prevent deadlocks

				// reset state attribute (after the log service has restarted)
				if totalState == stopped {
					totalState = 0
				}

				// start log service
				go s.run()
				// reply to the caller when the service has started
				// the go routine is necessary to prevent a deadlock; control must still be able to handle setServiceState messages
				go func() {
					for {
						// wait until the service is running
						if c.checkState(running) {
							break
						}
					}
					c.execServiceActionResponse <- signal{}
				}()
			case stop:
				// stop log service
				s.serviceStop <- signal{}
				// reply to the caller when the service has stopped
				// the go routine is necessary to prevent a deadlock; control must still be able to handle setServiceState messages
				go func() {
					for {
						// wait until the service is stopped
						if c.checkState(stopped) {
							break
						}
					}
					s.closeLogFile()
					c.execServiceActionResponse <- signal{}
				}()
			case initlog:
				logName := convertToString(s.attribute[logfilename])
				s.serviceConfig <- configMessage{initlog, logName}
			case newlog:
				newLogName := convertToString(s.attribute[logfilename])
				s.serviceConfig <- configMessage{newlog, newLogName}
			}
		case singleState = <-c.setServiceState:
			if singleState == stopped {
				// unset all other states
				totalState = singleState
			} else {
				// add new state to the total state
				totalState |= singleState
			}
		case state := <-c.checkServiceState:
			if totalState&state == state {
				c.checkServiceStateResponse <- true
			} else {
				c.checkServiceStateResponse <- false
			}
		}
	}
}

// service handles actions to be processed by the log service.
func (c *control) service(action int) signal {
	c.execServiceAction <- action
	return <-c.execServiceActionResponse
}

// checkState checks if the service has set the specified state.
func (c *control) checkState(state int) bool {
	c.checkServiceState <- state
	return <-c.checkServiceStateResponse
}

// setState sets the state of the log service.
func (c *control) setState(state int) {
	c.setServiceState <- state
}