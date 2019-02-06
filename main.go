package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
	// for config
	"github.com/BurntSushi/toml"
	"io/ioutil"

	expect "github.com/Netflix/go-expect"
)

var (
	config TestConfig

	configPath = flag.String("config", "config.toml", "Path to config")
)

type TestCase struct {
	Name    string `toml:"name"`
	Command string `toml:"command"`
	// regexp like in killer?
	ExpectedOutput string `toml:"expected_output"`
}

type TestConfig struct {
	Tests              []TestCase `toml:"tests"`
	Path               string     `toml:"path"`
	StopAfterFirstFail bool       `toml:"stop_after_first_fail"`
	// Millisecond
	Delay uint `toml:"delay"`
}

func main() {
	flag.Parse()
	var output bytes.Buffer
	configBytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		// TODO
		panic(err)
	}
	err = toml.Unmarshal(configBytes, &config)
	if err != nil {
		// TODO
		panic(err)
	}

	var delay time.Duration
	if config.Delay == 0 {
		config.Delay = 200
	}
	delay = time.Millisecond * time.Duration(config.Delay)

	var splited []string
	for i := range config.Tests {
		splited = strings.Fields(config.Tests[i].ExpectedOutput)
		sort.Strings(splited)
		config.Tests[i].ExpectedOutput = strings.Join(splited, " ")
	}

	// Init
	c, err := expect.NewConsole(expect.WithStdout(&output))
	//c, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	cmd := exec.Command("bash")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	go func() {
		c.ExpectEOF()
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second)
	c.SendLine("export PS1=\"#\"")
	c.SendLine("source " + config.Path)
	time.Sleep(delay)
	var completionOutput string
	var haveFailedTests int
	// tc aka test case
	for i, tc := range config.Tests {
		time.Sleep(delay)
		c.Send(tc.Command)
		time.Sleep(delay)

		output.Reset()
		// Tab Tab
		c.Send("\x09\x09")
		time.Sleep(delay)
		splited = []string{}
		for _, line := range strings.Split(output.String(), "\n") {
			// Skip void lines and not usefull to
			if strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimSpace(line)
			if len(line) == 1 {
				continue
			}

			splited = append(splited, strings.Fields(line)...)
		}
		sort.Strings(splited)
		completionOutput = strings.Join(splited, " ")
		if completionOutput != tc.ExpectedOutput {
			fmt.Printf("Test %d - %s - failed\n", i, tc.Name)
			fmt.Printf("We want \"%s\" but have \"%s\"\n", tc.ExpectedOutput, completionOutput)
			haveFailedTests++
			if config.StopAfterFirstFail {
				os.Exit(1)
			}
		} else {
			fmt.Printf("Test %d - %s - ok\n", i, tc.Name)
		}

		for i := 0; i < len(tc.Command); i++ {
			c.Send("\x08")
			c.Send("\x08")
		}

	}
	c.SendLine("exit")

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	if haveFailedTests > 0 {
		fmt.Println("We have some failed tests")
		os.Exit(1)
	}

}
