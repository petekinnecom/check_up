package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os/exec"
)

func Cmd(cmd string) bool {
	// taken from: http://stackoverflow.com/a/27764262
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return false
	}
	return true
}

func CheckUp(serviceName string, spec map[string]string, log func(string, int)) bool {
	up := Cmd(spec["command"])
	if up {
		log("up", 0)
		return true
	} else {
		log("down", 0)
		return false
	}

}

var data = `
services:
  up_service:
    command: exit 0
    retries: 1
    interval: 2
    timeout: 3
  down_service:
    command: exit 1
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
			fmt.Printf("%v'\n", msg)
		}
	}
}

func main() {
	yml := make(map[string]map[string]map[string]string)
	log := Logger(1)

	err := yaml.Unmarshal([]byte(data), &yml)
	if err != nil {
		panic("error")
	}

	for serviceName, spec := range yml["services"] {
		serviceLog := ServiceLogger(serviceName, log)
		CheckUp(serviceName, spec, serviceLog)
	}
}
