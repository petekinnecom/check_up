package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
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

func Logger(serviceName string) func(string) {
	return func(msg string) {
		fmt.Println("%v | '%v'\n", serviceName, msg)
	}
}

func main() {
	yml := make(map[string]map[string]map[string]string)

	err := yaml.Unmarshal([]byte(data), &yml)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	for name, test := range yml["services"] {
		isUp := Cmd(test["command"])
		fmt.Printf("service: %v, up? '%v'\n", name, isUp)
	}
}
