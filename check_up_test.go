package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"testing"
)

type LogLine struct {
	Msg string
	Level int
}

var logLines []LogLine

func log(s string, i int) {
	logLines = append(logLines, LogLine{s, i})
}

func resetLog(){
	logLines = make([]LogLine, 0)
}

func assertLog(msg string, i int, t *testing.T){
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
