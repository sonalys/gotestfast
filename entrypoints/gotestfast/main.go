package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sonalys/gotestfast"
)

func main() {
	projectFolder := flag.String("input", "", "Path to the Go project folder")
	outputFile := flag.String("output", "testlog.json", "Path to the output file")
	coverprofile := flag.String("coverprofile", "", "If set, writes a coverage profile to the given file")
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

	tests, err := gotestfast.ScanTests(*projectFolder, *pkg)
	if err != nil {
		log.Panic().Err(err).Msg("failed to get tests")
	}

	switch gotestfast.RunAndSave(*projectFolder, tests, *outputFile, *coverprofile) {
	case nil:
		return
	case gotestfast.ErrTestFailed:
		os.Exit(1)
	default:
		log.Panic().Err(err).Msg("failed to execute and persist tests")
	}
}
