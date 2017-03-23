package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os/exec"
	"strconv"
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
	log := ServiceLogger(service.Name, unboundLog)
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

func waitAll(services []Service, log func(string, int)) {
	allUp := false

	for !allUp {
		allUp = checkAll(services, log)
		if !allUp {
			log("retrying check up", 1)
		}
	}
}

func loadFile(filePath string) []Service {
	yml := make(map[string]map[string]map[string]string)

	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("couldn't read file")
	}

	err = yaml.Unmarshal([]byte(fileContents), &yml)
	if err != nil {
		panic("error")
	}

	services := make([]Service, 0)
	for name, spec := range yml["services"] {
		timeout, err := strconv.Atoi(spec["timeout"])
		if err != nil {
			timeout = 3
		}
		retries, err := strconv.Atoi(spec["retries"])
		if err != nil {
			retries = 0
		}
		interval, err := strconv.Atoi(spec["interval"])
		if err != nil {
			interval = 1
		}
		services = append(services,
			Service{
				Name:     name,
				Timeout:  timeout,
				Command:  spec["command"],
				Retries:  retries,
				Interval: interval})
	}
	return services
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

func main() {
	filePathPtr := flag.String("file", "check_up.yml", "path to configuration yml")
	waitPtr := flag.Bool("wait", false, "check services repeatedly until all are up")
	verbosePtr := flag.Bool("verbose", false, "output more info")

	flag.Parse()

	logLevel := 0
	if *verbosePtr {
		logLevel = 1
	}

	log := Logger(logLevel)
	services := loadFile(*filePathPtr)
	services = filterServices(services, flag.Args())

	if *waitPtr {
		waitAll(services, log)
	} else {
		checkAll(services, log)
	}
}
