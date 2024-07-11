package gotestfast

import (
	"encoding/json"
	"os"
)

func RearrangeTestRecords(previous, current []TestRecord) []TestRecord {
	uniqueKeys := make(map[string]bool, len(previous))
	for _, record := range previous {
		key := record.Package + "/" + record.Name
		uniqueKeys[key] = true
	}
	result := make(TestRecords, 0, len(previous)+len(current))
	for _, record := range current {
		key := record.Package + "/" + record.Name
		if !uniqueKeys[key] {
			result = append(result, record)
		}
	}
	result = append(result, previous...)
	return result
}

func ReadRecordFile(filePath string) ([]TestRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	var records TestRecords
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&records); err != nil {
		return nil, err
	}
	return records, nil
}
