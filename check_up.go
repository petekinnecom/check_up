package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Service struct {
	Name     string
	Command  string
	Timeout  int
	Retries  int
	Interval int
}

type YamlSpec struct {
	Services []Service
}

func execWithTimeout(command string, timeout int, log func(string, int)) bool {
	// taken from: http://stackoverflow.com/a/27764262
	log(fmt.Sprintf("%v", command), 1)
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

func CheckUp(service Service, unboundLog func(string, int)) bool {
	log := func(msg string, level int) {
		message := fmt.Sprintf("%v | %v", service.Name, msg)
		unboundLog(message, level)
	}

	for i := 0; i <= service.Retries; i++ {
		log("trying", 1)
		up := execWithTimeout(service.Command, service.Timeout, log)
		if up {
			log("up", 1)
			return true
		} else if i < service.Retries {
			log(fmt.Sprintf("sleep %v interval", service.Interval), 1)
			time.Sleep(time.Duration(service.Interval) * time.Second)
		}
	}

	log("down", 0)
	return false
}

func Logger(logLevel int) func(string, int) {
	return func(msg string, msgLevel int) {
		if msgLevel <= logLevel {
			fmt.Printf("%v\n", msg)
		}
	}
}

func checkAll(services []Service, log func(string, int)) bool {
	var wg sync.WaitGroup
	wg.Add(len(services))

	var allUp bool
	allUp = true
	for _, service := range services {
		go func(service Service) {
			defer wg.Done()

			// Some crazy business here
			// Don't know why this works with the extra assignment...
			// Probably thread safety issue
			up := CheckUp(service, log)
			allUp = allUp && up
		}(service)
	}
	wg.Wait()

	return allUp
}

func waitAll(services []Service, log func(string, int)) bool {
	allUp := false

	for !allUp {
		allUp = checkAll(services, log)
		if !allUp {
			log("retrying check up", 1)
			time.Sleep(1 * time.Second)
		}
	}
	return true
}

func loadFile(filePath string, log func(string, int)) YamlSpec {
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("couldn't read file")
	}

	yamlSpec := YamlSpec{}
	err = yaml.Unmarshal([]byte(fileContents), &yamlSpec)
	if err != nil {
		panic("Invalid yml file")
	}
	return yamlSpec
}

func filterServices(services []Service, serviceNames []string) []Service {
	if len(serviceNames) == 0 {
		return services
	} else {
		selectedServices := make([]Service, 0)

		for _, name := range serviceNames {
			for _, service := range services {
				if service.Name == name {
					selectedServices = append(selectedServices, service)
				}
			}
		}

		return selectedServices
	}
}

func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawService Service
	raw := rawService{Timeout: 2, Retries: 0, Interval: 1} // Put your defaults here
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*s = Service(raw)
	return nil
}

func LoadServices(filePath string, serviceNames []string, log func(string, int)) []Service {
	yamlSpec := loadFile(filePath, log)
	return filterServices(yamlSpec.Services, serviceNames)
}

func cliStart(serviceNames []string, logLevel int, wait bool, filePath string) bool {
	log := Logger(logLevel)
	services := LoadServices(filePath, serviceNames, log)

	if wait {
		return waitAll(services, log)
	} else {
		return checkAll(services, log)
	}
}

func main() {
	filePathPtr := flag.String("file", "check_up.yml", "path to configuration yml")
	waitPtr := flag.Bool("wait", false, "check services repeatedly until all are up")
	verbosePtr := flag.Bool("verbose", false, "output more info")
	flag.Parse()

	logLevel := 0
	if *verbosePtr {
		logLevel = 1
	}

	allUp := cliStart(flag.Args(), logLevel, *waitPtr, *filePathPtr)
	if allUp {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
