package gotestfast

import (
	"encoding/json"
	"os"
	"sort"
)

type (
	ActionType string

	Test struct {
		Name    string `json:"Output"`
		Package string `json:"Package"`
		Action  string `json:"Action"`
	}

	Tests []Test

	TestRecord struct {
		Package string `json:"package"`
		Name    string `json:"name"`
		Passed  bool   `json:"passed"`
		details string `json:"-"`
	}

	TestRecords []TestRecord
)

const (
	ActionTypeOutput = "output"
	ActionTypePass   = "pass"
)

func (records TestRecords) WriteToFile(filePath string) error {
	records.Sort()
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(records); err != nil {
		return err
	}

	return nil
}

func (records TestRecords) Sort() {
	sort.SliceStable(records, func(i, j int) bool {
		return !records[i].Passed && records[j].Passed
	})
}

func (tests Tests) ToRecord() []TestRecord {
	var records []TestRecord
	for _, test := range tests {
		record := TestRecord{
			Package: test.Package,
			Name:    test.Name,
			Passed:  false,
		}
		records = append(records, record)
	}
	return records
}
