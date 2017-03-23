package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os/exec"
)

type Service struct {
	A string
	B struct {
		RenamedC int   `yaml:"c"`
		D        []int `yaml:",flow"`
	}
}

func Cmd(cmd string) []byte {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Printf("err: %v", err)
		panic("some error found")
	}
	return out
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

func main() {
	yml := make(map[string]map[string]map[string]string)

	err := yaml.Unmarshal([]byte(data), &yml)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	for name, test := range yml["services"] {
		fmt.Printf("service: %v runs '%v'\n", name, test["command"])
	}
	out := Cmd("exit 0")
	fmt.Printf("output: %v", out)
}
