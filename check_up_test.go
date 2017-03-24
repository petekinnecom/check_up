package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"testing"
	"time"
)

type LogLine struct {
	Msg   string
	Level int
}

var logLines []LogLine

func log(s string, i int) {
	// fmt.Printf("log: %v\n", s)
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

func TestCheckAll__allUp(t *testing.T) {
	resetLog()
	services := make([]Service, 0)

	services = append(services, Service{
		Name:     "service_1",
		Command:  "exit 0",
		Timeout:  1,
		Retries:  0,
		Interval: 0})

	services = append(services, Service{
		Name:     "service_2",
		Command:  "exit 0",
		Timeout:  1,
		Retries:  0,
		Interval: 0})

	allUp := checkAll(services, log)
	if !allUp {
		t.Error("All services should report up")
	}
}

func TestCheckAll__someDown(t *testing.T) {
	resetLog()
	services := make([]Service, 0)

	services = append(services, Service{
		Name:     "service_1",
		Command:  "exit 0",
		Timeout:  1,
		Retries:  0,
		Interval: 0})

	services = append(services, Service{
		Name:     "service_2",
		Command:  "exit 1",
		Timeout:  1,
		Retries:  0,
		Interval: 0})

	allUp := checkAll(services, log)
	if allUp {
		t.Error("One service down should make it false")
	}
}

func TestCheckAll__concurrency(t *testing.T) {
	resetLog()
	services := make([]Service, 0)

	services = append(services, Service{
		Name:     "service_1",
		Command:  "sleep 1",
		Timeout:  3,
		Retries:  0,
		Interval: 0})

	services = append(services, Service{
		Name:     "service_2",
		Command:  "sleep 1",
		Timeout:  3,
		Retries:  0,
		Interval: 0})
	start := time.Now()
	allUp := checkAll(services, log)
	if !allUp {
		t.Error("All services are up, this should return true")
	}
	elapsed := time.Since(start)

	if elapsed >= (time.Duration(2) * time.Second) {
		t.Error("checks should run concurrently")
	}
}

func TestLoadServices__all(t *testing.T) {
	services := LoadServices("./test/check_up.yml", []string{}, log)

	assertServiceNames([]string{"service_0", "service_1", "service_2"}, services, t)
}

func TestLoadServices__defaultValues(t *testing.T) {
	services := LoadServices("./test/check_up.yml", []string{"service_2"}, log)

	if services[0].Timeout != 2 {
		t.Error("Timeout should default to 2s but got: ", services[0].Timeout)
	}
}

func TestLoadServices__specified(t *testing.T) {
	serviceNames := []string{"service_0", "service_2"}
	services := LoadServices("./test/check_up.yml", serviceNames, log)

	assertServiceNames(serviceNames, services, t)
}

func assertServiceNames(expectedNames []string, services []Service, t *testing.T) {
	names := []string{}
	for _, s := range services {
		names = append(names, s.Name)
	}
	sort.Strings(expectedNames)
	sort.Strings(names)

	for i := range expectedNames {
		if expectedNames[i] != names[i] {
			t.Error("Incorrectly loaded services\n expected: ", expectedNames, "\ngot: ", names)
		}
	}
}
