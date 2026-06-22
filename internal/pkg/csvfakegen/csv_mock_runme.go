package csvfakegen

import "log"

func CreateFakeData(path string, records int) {
	if path == "" {
		path = "./random_data.csv"
	}
	generator := NewCsvMockGenerator(path, records)
	if err := generator.Generate(); err != nil {
		log.Fatalf("Critacal error: %v. Can't continue.", err)
	}
}