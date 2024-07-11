package gotestfast

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

var Logger = log.Logger

var ErrTestFailed = fmt.Errorf("test failed")

func RunAndSave(projectFolder string, tests Tests, filePath string) error {
	log.Info().Msgf("Scanned %d tests", len(tests))
	current := tests.ToRecord()

	// Parse existing test records from file
	previousRecords, err := ReadRecordFile(filePath)
	if err != nil {
		return err
	}
	log.Info().Msgf("Loaded %d previous test records", len(previousRecords))

	testExecutionOrder := RearrangeTestRecords(previousRecords, current)
	log.Info().Msgf("Executing %d tests", len(testExecutionOrder))

	updatedRecords, err := Run(projectFolder, testExecutionOrder)
	if err := updatedRecords.WriteToFile(filePath); err != nil {
		return err
	}
	return err
}

func Run(projectFolder string, tests TestRecords) (TestRecords, error) {
	for i := range tests {
		cur := runTest(projectFolder, tests[i])
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

func runTest(projectFolder string, test TestRecord) TestRecord {
	cmd := exec.Command("go", "test", "-run", test.Name, test.Package)
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
