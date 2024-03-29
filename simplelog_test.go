package simplelog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestStartup(t *testing.T) {
	logFile := "test1.log"
	Startup(1)

	if a := s.isActive(); a != true {
		t.Error("Expected state true but got", a)
	} else {
		s.stop(false)
		s.setActive(false)
	}
	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}
}

func TestShutdown(t *testing.T) {
	logFile := "test1.log"
	Startup(1)
	Shutdown(false)

	if a := s.isActive(); a == true {
		t.Error("Expected state false but got", a)
		s.stop(false)
		s.setActive(false)
	}
	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}
}

func TestChangeLogFile(t *testing.T) {
	logFile1 := "test1.log"
	logFile2 := "test2.log"
	filePerms := "-rw-r--r--"
	fileSize := 0

	if _, err := os.Stat(logFile1); err == nil {
		os.Remove(logFile1)
	}
	if _, err := os.Stat(logFile2); err == nil {
		os.Remove(logFile2)
	}

	Startup(1)
	SetupLog(logFile1, false)
	SwitchLog(logFile2)
	Shutdown(false)

	data, err := os.Stat(logFile1)
	if err != nil {
		t.Error("Expected to find file", logFile1, "- but got:", err)
	} else if data.Mode().String() != filePerms {
		t.Error("Expected file permissions", filePerms, "but found:", data.Mode().String())
	} else if data.Size() != 0 {
		t.Error("Expected file size", fileSize, "but found:", data.Size())
	} else {
		os.Remove(logFile1)
	}

	data, err = os.Stat(logFile2)
	if err != nil {
		t.Error("Expected to find file", logFile2, "- but got:", err)
	} else if data.Mode().String() != filePerms {
		t.Error("Expected file permissions", filePerms, "but found:", data.Mode().String())
	} else if data.Size() != 0 {
		t.Error("Expected file size", fileSize, "but found:", data.Size())
	} else {
		os.Remove(logFile2)
	}
}

func TestSetPrefix(t *testing.T) {
	s = new(simpleLogService) // reset service instance
	logFile := "test1.log"
	expectedPrefix := "#2006-01-02 15:04:05.000000#[Test]"

	Startup(1)
	SetPrefix(STDOUT, "#2006-01-02 15:04:05.000000#", "[Test]")
	SetPrefix(FILE, "#2006-01-02 15:04:05.000000#", "[Test]")
	Shutdown(false)

	var prefix string
	for _, v := range s.stdoutLogger.prefix {
		prefix += v
	}
	if !strings.Contains(prefix, expectedPrefix) {
		t.Error("Expected to find:", expectedPrefix, "- but found:", prefix)
	}

	prefix = ""
	for _, v := range s.fileLogger.prefix {
		prefix += v
	}
	if !strings.Contains(prefix, expectedPrefix) {
		t.Error("Expected to find:", expectedPrefix, "- but found:", prefix)
	}
	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}
}

func TestLogToStdout(t *testing.T) {
	s = new(simpleLogService) // reset service instance
	stdOut := os.Stdout
	logFile := "test1.log"

	r, w, _ := os.Pipe()
	os.Stdout = w

	Startup(1)
	Write(STDOUT, "The answer to all questions is", 42)
	Shutdown(false)

	_ = w.Close()

	result, _ := io.ReadAll(r)
	output := string(result)

	os.Stdout = stdOut

	if !strings.Contains(output, "The answer to all questions is "+fmt.Sprint(42)) {
		t.Error("Expected to find:", "The answer to all questions is "+fmt.Sprint(42), "- but found:", output)
	}
	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}
}

func TestLogToFile(t *testing.T) {
	s = new(simpleLogService) // reset service instance
	logFile := "test1.log"

	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}

	Startup(1)
	SetupLog(logFile, false)
	Write(FILE, "The answer to all questions is", 42)
	Shutdown(false)

	data, err := os.ReadFile(logFile)

	if err != nil {
		t.Error("Expected to find file", logFile, "- but got:", err)
	} else if !strings.Contains(string(data), "The answer to all questions is "+fmt.Sprint(42)) {
		t.Error("Expected log record contains:", "The answer to all questions is "+fmt.Sprint(42), "- but it doesn't:", string(data))
	} else {
		os.Remove(logFile)
	}
}

func TestConditionalLogToFile(t *testing.T) {
	s = new(simpleLogService) // reset service instance
	logFile := "test1.log"

	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}

	Startup(1)
	SetupLog(logFile, false)
	ConditionalWrite(false, FILE, "The answer to all questions is", 42)
	Shutdown(false)

	data, err := os.ReadFile(logFile)

	if err != nil {
		t.Error("Expected to find file", logFile, "- but got:", err)
	} else if string(data) != "" {
		t.Error("Expected an empty file - but it contains:", string(data))
	} else {
		os.Remove(logFile)
	}
}

func TestLogToMulti(t *testing.T) {
	s = new(simpleLogService) // reset service instance
	stdOut := os.Stdout
	logFile := "test1.log"

	r, w, _ := os.Pipe()
	os.Stdout = w

	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}

	Startup(1)
	SetupLog(logFile, false)
	Write(MULTI, "The answer to all questions is", 42)
	Shutdown(false)

	_ = w.Close()

	result, _ := io.ReadAll(r)
	output := string(result)

	os.Stdout = stdOut

	// check output sent to stdout
	if !strings.Contains(output, "The answer to all questions is "+fmt.Sprint(42)) {
		t.Error("Expected to find:", "The answer to all questions is "+fmt.Sprint(42), "- but found:", output)
	}

	// check output sent to file
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Error("Expected to find file", logFile, "- but got:", err)
	} else if !strings.Contains(string(data), "The answer to all questions is "+fmt.Sprint(42)) {
		t.Error("Expected log record:", "The answer to all questions is "+fmt.Sprint(42), "- but got:", string(data))
	} else {
		os.Remove(logFile)
	}
}

func BenchmarkLog(b *testing.B) {
	s = new(simpleLogService) // reset service instance
	logFile := "test1.log"

	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}

	Startup(1)
	SetupLog(logFile, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Write(FILE, "The answer to all questions is", 42)
	}
	Shutdown(false)

	if _, err := os.Stat(logFile); err == nil {
		os.Remove(logFile)
	}
}
