package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

type LogLine struct {
	Msg   string
	Level int
}

var logLines []LogLine

func log(s string, i int) {
	logLines = append(logLines, LogLine{s, i})
}

func resetLog() {
	logLines = make([]LogLine, 0)
}

func randomIdentifier() string {
	now := time.Now().Unix()
	return fmt.Sprintf("%v", now)
}

func assertLog(msg string, i int, t *testing.T) {
	for _, logLine := range logLines {
		if logLine.Msg == msg && logLine.Level == i {
			return
		}
	}
	t.Error(fmt.Sprintf("Did not log msg: '%v' level: '%v'. Found logs: %v", msg, i, logLines))
}

func TestExecWithTimeout__Success(t *testing.T) {
	resetLog()
	r := execWithTimeout("exit 0", 3, log)
	if !r {
		t.Error("exit status 0 should return true")
	}
	assertLog("exit 0", 1, t)
}

func TestExecWithTimeout__Failure(t *testing.T) {
	r := execWithTimeout("exit 1", 3, log)
	if r {
		t.Error("non-zero exits should return false")
	}
	assertLog("exit 1", 1, t)
	assertLog("exit status 1", 1, t)
}

func TestExecWithTimeout__Timeout(t *testing.T) {
	logLines = make([]LogLine, 0)
	r := execWithTimeout("sleep 2", 1, log)
	if r {
		t.Error("timeouts should return false")
	}
	assertLog("timed out after 1 seconds", 1, t)
}

func TestExecWithTimeout__Timeout__ShouldKillProcess(t *testing.T) {
	cmd := "sleep 10; random_identifier__1234567890"
	execWithTimeout(cmd, 1, log)

	outputBytes, err := exec.Command("bash", "-c", "ps aux | grep -i random_identifier__1234567890").Output()
	if err != nil {
		t.Error("Failed to grep for running process %v", err)
	}

	output := fmt.Sprintf("%s", outputBytes)
	matched, err := regexp.MatchString(cmd, output)
	if err != nil {
		t.Error("Failed to perform regex")
	}
	if matched {
		t.Error("Sleep command should have been killed")
	}
}

func TestCheckUp__up(t *testing.T) {
	resetLog()
	service := Service{
		Name:     "serviceName",
		Command:  "exit 0",
		Retries:  0,
		Timeout:  1,
		Interval: 1}
	up := CheckUp(service, log)
	if !up {
		t.Error("should be up")
	}
	assertLog("serviceName | up", 1, t)
}

func TestCheckUp__down(t *testing.T) {
	resetLog()
	service := Service{
		Name:     "serviceName",
		Command:  "exit 1",
		Retries:  0,
		Timeout:  1,
		Interval: 1}
	up := CheckUp(service, log)
	if up {
		t.Error("should be down")
	}
	assertLog("serviceName | down", 0, t)
}

func TestCheckUp__retry(t *testing.T) {
	resetLog()
	filePath := randomIdentifier()
	service := Service{
		Name:     "serviceName",
		Command:  fmt.Sprintf("(test -f /tmp/%v && rm /tmp/%v) || (touch /tmp/%v && exit 1)", filePath, filePath, filePath),
		Retries:  1,
		Timeout:  1,
		Interval: 1}
	up := CheckUp(service, log)
	if !up {
		t.Error("should be up")
	}
	assertLog("serviceName | sleep 1 interval", 1, t)
	assertLog("serviceName | up", 1, t)
}
