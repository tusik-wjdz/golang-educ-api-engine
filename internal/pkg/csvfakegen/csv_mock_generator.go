package csvfakegen

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// CsvMockGenerator conf. struct.
type CsvMockGenerator struct {
	FilePath   string
	NumRecords int
}

// Ctor
func NewCsvMockGenerator(path string, records int) *CsvMockGenerator {
	return &CsvMockGenerator{
		FilePath:   path,
		NumRecords: records,
	}
}

// main generate (or shitstorm ;)) func
func (g *CsvMockGenerator) Generate() error {
	// open file
	file, err := os.Create(g.FilePath)
	if err != nil {
		return fmt.Errorf("Unable to create file: %v", err)
	}
	defer file.Close()

	// init writer (def. RAM buffering - extremly fast)
	writer := csv.NewWriter(file)
	defer writer.Flush() // Buffer must be dumped to file (hdd) before close 

	// write cols. headers
	headers := []string{"name", "price", "qty", "description", "color"}
	if err := writer.Write(headers); err != nil {
		return err
	}	
	// init seed rand
	gofakeit.Seed(time.Now().UnixNano())

	log.Printf("Starting generating %d records to (csv) file: %s...\n", g.NumRecords, g.FilePath)
	startTime := time.Now()

	// main seed loop
	for i := 1; i <= g.NumRecords; i++ {
		name := gofakeit.ProductName()		
		// rand. and format price (10 to 5000)
		price := fmt.Sprintf("%.2f", gofakeit.Price(10.0, 5000.0))
		qty := strconv.Itoa(gofakeit.Number(1, 500))
		color := gofakeit.Color()

		var desc string
		// shitstorm desc with every 15 rows
		if i%15 == 0 {			
			// args: paragraphs, sentences, words, sep
			desc = gofakeit.Paragraph(3, 5, 12, "\n")
		} else {
			desc = gofakeit.Sentence(6) // short desc. (about six words)
		}		
		// save single row into buffer
		if err := writer.Write([]string{name, price, qty, desc, color}); err != nil {
			return fmt.Errorf("Unable to write row %d: %v. Can't continue", i, err)
		}

		if i%100000 == 0 {
			log.Printf("Generated %d / %d records...", i, g.NumRecords)
		}
	}

	log.Printf("Success! File has been created in: %v", time.Since(startTime))
	return nil
}
