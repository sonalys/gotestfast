package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sonalys/gotestfast"
)

func main() {
	projectFolder := flag.String("input", "", "Path to the Go project folder")
	outputFile := flag.String("output", "testlog.json", "Path to the output file")
	jsonOutput := flag.Bool("json", false, "Output test results as JSON")
	pkg := flag.String("pkg", "./...", "Comma separated list of packages to test")
	flag.Parse()

	if !*jsonOutput {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out: os.Stdout,
		})
		gotestfast.Logger = log.Logger
	}

	cmd := exec.Command("go", "test", "-json", "-list", ".", *pkg)
	cmd.Dir = *projectFolder

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get stderr pipe")
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get stdout pipe")
	}

	if err = cmd.Start(); err != nil {
		log.Panic().Err(err).Msg("failed to start command")
	}

	tests, err := parseTests(stdOut)
	if err != nil {
		log.Panic().Err(err).Msg("failed to parse tests")
	}

	if err = cmd.Wait(); err != nil {
		log.Panic().Err(err).Msg("failed to wait for command")
	}

	if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
		errorOutput, _ := io.ReadAll(stdErr)
		log.Panic().Str("error", string(errorOutput)).Int("exitCode", exitCode).Msg("failed to list tests")
	}

	if err := gotestfast.RunAndSave(*projectFolder, tests, *outputFile); err != nil {
		if err == gotestfast.ErrTestFailed {
			os.Exit(1)
			return
		}
		log.Panic().Err(err).Msg("failed to execute and persist tests")
	}
}

func parseTests(stdOut io.Reader) ([]gotestfast.Test, error) {
	var tests []gotestfast.Test
	reader := bufio.NewReader(stdOut)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		var test gotestfast.Test
		if err := json.Unmarshal(line, &test); err != nil {
			return nil, err
		}
		if test.Action != gotestfast.ActionTypeOutput || !strings.HasPrefix(test.Name, "Test_") {
			continue
		}
		test.Name = strings.ReplaceAll(test.Name, "\n", "")
		tests = append(tests, test)
	}
	return tests, nil
}
