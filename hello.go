package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

func Cmd(command string, timeout int, log func(string, int)) bool {
	// taken from: http://stackoverflow.com/a/27764262
	log(fmt.Sprintf("`%v`", command), 1)
	cmd := exec.Command("bash", "-c", command)

	cmd.Start()
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			panic(fmt.Sprintf("failed to kill: %v", err))
		}
		log(fmt.Sprintf("timed out after %v seconds", timeout), 1)
		return false
	case err := <-done:
		if err != nil {
			log(fmt.Sprintf("%v", err), 1)
			return false
		} else {
			return true
		}
	}
}

func CheckUp(serviceName string, spec map[string]string, unboundLog func(string, int)) bool {
	log := ServiceLogger(serviceName, unboundLog)
	timeout, err := strconv.Atoi(spec["timeout"])
	if err != nil {
		panic("failed to parse timeout")
	}

	up := Cmd(spec["command"], timeout, log)
	if up {
		log("up", 1)
		return true
	} else {
		log("down", 0)
		return false
	}

}

var data = `
services:
  up_service:
    command: sleep 2
    retries: 1
    interval: 2
    timeout: 3
  down_service:
    command: test -f /tmp/pass
    retries: 1
    interval: 2
    timeout: 3
  other_service:
    command: sleep 1
    timeout: 3
`

func ServiceLogger(serviceName string, logger func(string, int)) func(string, int) {
	return func(msg string, level int) {
		message := fmt.Sprintf("%v | %v", serviceName, msg)
		logger(message, level)
	}
}

func Logger(logLevel int) func(string, int) {
	return func(msg string, msgLevel int) {
		if msgLevel <= logLevel {
			fmt.Printf("%v\n", msg)
		}
	}
}

func checkAll(servicesYml map[string]map[string]string, log func(string, int)) bool {
	var wg sync.WaitGroup
	wg.Add(len(servicesYml))

	results := make(map[string]bool)

	for serviceName, spec := range servicesYml {
		go func(serviceName string, spec map[string]string) {
			defer wg.Done()
			results[serviceName] = CheckUp(serviceName, spec, log)
		}(serviceName, spec)
	}
	wg.Wait()

	var allUp bool
	allUp = true
	for _, isUp := range results {
		allUp = allUp && isUp
	}

	return allUp
}

func waitAll(servicesYml map[string]map[string]string, log func(string, int)) {
	allUp := false

	for !allUp {
		allUp = checkAll(servicesYml, log)
		if !allUp {
			log("check again", 0)
		}
	}
}

func main() {
	yml := make(map[string]map[string]map[string]string)
	log := Logger(0)

	err := yaml.Unmarshal([]byte(data), &yml)
	if err != nil {
		panic("error")
	}

	waitAll(yml["services"], log)
}
