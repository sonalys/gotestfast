package gotestfast

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

var Logger = log.Logger

var ErrTestFailed = fmt.Errorf("test failed")

func readGoTestList(stdOut io.Reader) (Tests, error) {
	var tests Tests
	reader := bufio.NewReader(stdOut)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		var test Test
		if err := json.Unmarshal(line, &test); err != nil {
			return nil, err
		}
		if test.Action != ActionTypeOutput || !strings.HasPrefix(test.Name, "Test_") {
			continue
		}
		test.Name = strings.ReplaceAll(test.Name, "\n", "")
		tests = append(tests, test)
	}
	return tests, nil
}

func RunAndSave(projectFolder string, tests Tests, filePath string, coverProfile string) error {
	log.Info().Msgf("Scanned %d tests", len(tests))
	current := tests.ToRecord()

	// Parse existing test records from file
	previousRecords, err := ReadRecordFile(filePath)
	if err != nil {
		return err
	}
	log.Info().Msgf("Loaded %d previous test records", len(previousRecords))

	testExecutionOrder := rearrangeTestRecords(previousRecords, current)
	log.Info().Msgf("Executing %d tests", len(testExecutionOrder))

	updatedRecords, err := Run(projectFolder, testExecutionOrder, coverProfile)
	if err := updatedRecords.WriteToFile(filePath); err != nil {
		return err
	}
	return err
}

func Run(projectFolder string, tests TestRecords, coverProfile string) (TestRecords, error) {
	if coverProfile != "" {
		os.Remove(coverProfile)
	}
	for i := range tests {
		cur := runTest(projectFolder, tests[i], coverProfile)
		tests[i] = cur
		if cur.Passed {
			Logger.Info().Str("package", tests[i].Package).Str("name", tests[i].Name).Str("result", getTestResultString(cur.Passed)).Send()
			continue
		}
		Logger.Error().Str("package", tests[i].Package).Str("name", tests[i].Name).Str("result", getTestResultString(cur.Passed)).Send()
		fmt.Fprintf(os.Stderr, cur.details)
		return tests, ErrTestFailed
	}
	return tests, nil
}

func appendToCoverProfile(coverProfile string, tmpFile string) {
	file, err := os.Open(tmpFile)
	if err != nil {
		log.Panic().Err(err).Msg("failed to open tmpFile")
	}
	defer file.Close()
	defer os.Remove(tmpFile)

	targetFile, err := os.OpenFile(coverProfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panic().Err(err).Msg("failed to open target file")
	}
	defer targetFile.Close()

	stat, err := targetFile.Stat()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get stat of target file")
	}

	scanner := bufio.NewScanner(file)
	if stat.Size() != 0 {
		scanner.Scan()
	}

	for scanner.Scan() {
		line := scanner.Text()
		_, err = fmt.Fprintln(targetFile, line)
		if err != nil {
			log.Panic().Err(err).Msg("failed to append content to target file")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Panic().Err(err).Msg("failed to read tmpFile")
	}
}

func runTest(projectFolder string, test TestRecord, coverProfile string) TestRecord {
	var cmd *exec.Cmd
	tmpFile := os.TempDir() + "/coverage.out"
	if coverProfile != "" {
		cmd = exec.Command("go", "test", "-coverprofile", tmpFile, "-run", test.Name, test.Package)
		defer appendToCoverProfile(coverProfile, tmpFile)
	} else {
		cmd = exec.Command("go", "test", "-run", test.Name, test.Package)
	}
	cmd.Dir = projectFolder

	stdErr, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get stderr pipe")
	}

	if err = cmd.Start(); err != nil {
		log.Panic().Err(err).Msg("failed to start command")
	}

	errorOutput, err := io.ReadAll(stdErr)
	if err != nil {
		log.Panic().Err(err).Msg("failed to read error output")
	}

	if err = cmd.Wait(); err != nil && err.Error() != "exit status 1" {
		log.Panic().Err(err).Msg("failed to wait for command")
	}

	return TestRecord{
		Package: test.Package,
		Name:    test.Name,
		Passed:  cmd.ProcessState.ExitCode() == 0,
		details: string(errorOutput),
	}
}

func getTestResultString(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}

func ScanTests(projectFolder, pkg string) (Tests, error) {
	cmd := exec.Command("go", "test", "-json", "-list", ".", pkg)
	cmd.Dir = projectFolder
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	tests, err := readGoTestList(stdOut)
	if err != nil {
		return nil, fmt.Errorf("failed to read tests: %w", err)
	}
	if err = cmd.Wait(); err != nil {
		return nil, fmt.Errorf("failed to wait for command: %w", err)
	}
	if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
		errorOutput, _ := io.ReadAll(stdErr)
		return nil, fmt.Errorf("failed to run test list: %s", errorOutput)
	}
	return tests, nil
}
