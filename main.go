package simplelog

import (
	"fmt"
	"runtime"
)

func WriteToStdout(prefix string, values ...any) {
	if sLog.ServiceRunState() {
		logMessage := assembleToString(values)
		ld := message{STDOUT, prefix, logMessage}
		sLog.data <- ld
	} else {
		// TODO: add panic call
		fmt.Println("Log service has not been started.")
	}
}

func WriteToFile(prefix string, values ...any) {
	if sLog.ServiceRunState() {
		logMessage := assembleToString(values)
		ld := message{FILE, prefix, logMessage}
		sLog.data <- ld
	} else {
		// TODO: add panic call
		fmt.Println("Log service has not been started.")
	}
}

func WriteToMultiple(prefix string, values ...any) {
	if sLog.ServiceRunState() {
		logMessage := assembleToString(values)
		ld := message{MULTI, prefix, logMessage}
		sLog.data <- ld
	} else {
		// TODO: add panic call
		fmt.Println("Log service has not been started.")
	}
}

func StartService(msgBuffer int) {
	if !sLog.ServiceRunState() {
		sLog.initialize(msgBuffer)
		go func() {
			defer close(sLog.data)
			defer close(sLog.stopService)
			defer sLog.fileHandle.Close()

			for {
				select {
				case logMessage := <-sLog.data:
					switch logMessage.target {
					case STDOUT:
						sLog.Logger(STDOUT, logMessage.prefix).Print(logMessage.data)
					case FILE:
						sLog.Logger(FILE, logMessage.prefix).Print(logMessage.data)
					case MULTI:
						sLog.Logger(STDOUT, logMessage.prefix).Print(logMessage.data)
						sLog.Logger(FILE, logMessage.prefix).Print(logMessage.data)
					}
				case <-sLog.stopService:
					sLog.SetServiceRunState(false)
					return
				}
			}
		}()
	} else {
		_, filename, line, _ := runtime.Caller(1)
		errMsg := fmt.Sprintf("Log service was already started - %s: %d", filename, line)
		panic(errMsg)
	}
}

func StopService() {
	if sLog.ServiceRunState() {
		// wait until all messages have been logged by the service
		for len(sLog.data) > 0 {
			continue
		}
		// all messages are logged - the service can be stopped gracefully
		sLog.stopService <- trigger{}
	}
	// TODO: add panic call
}
