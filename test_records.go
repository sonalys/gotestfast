package gotestfast

import (
	"encoding/json"
	"os"
)

func rearrangeTestRecords(previous, current TestRecords) TestRecords {
	uniqueKeys := make(map[string]struct{}, len(previous))
	for _, record := range previous {
		key := record.Package + "/" + record.Name
		uniqueKeys[key] = struct{}{}
	}
	result := make(TestRecords, 0, len(previous)+len(current))
	for _, record := range current {
		key := record.Package + "/" + record.Name
		if _, ok := uniqueKeys[key]; !ok {
			result = append(result, record)
		}
	}
	result = append(result, previous...)
	return result
}

func ReadRecordFile(filePath string) (TestRecords, error) {
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
