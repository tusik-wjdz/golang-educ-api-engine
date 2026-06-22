package tools

import (
    "encoding/csv"
    "fmt"
    "io"
    "os"
)

type CsvReader struct {
    // required columns in target file (should be set before start)
    ReqColumns 			[]string
    // col. counter
    ColsInRow			int
}

// ctor
func GetReader() *CsvReader {
    return &CsvReader{}
}

// col. checker (agains passed array of expected columns in file)
func (cr *CsvReader) checkColumns(header []string) error {
    cr.ColsInRow = len(cr.ReqColumns)
    if cr.ColsInRow != len(header) {
        // well... looks like it doesn't match
        return fmt.Errorf(
            "Invalid number of columns. Expected: %d, given: %d",
            cr.ColsInRow,
            len(header),
        )
    }
    // based on indexes
    for i := range cr.ReqColumns {
        if header[i] != cr.ReqColumns[i] {
            return fmt.Errorf("There is no column %s in CSV header.", cr.ReqColumns[i])
        }
    }
    // well ...looks ok
    return nil
}

// main method for read data (cab be us like this for row := range ReadCsv(path) {})
func (cr *CsvReader) ReadCsv(path string) (<-chan []string, error) {
    // try open file
    inpf, err 	:= os.Open(path)
    if err != nil {
        return nil, err
    }
    // create reader
    reader 		:= csv.NewReader(inpf)
    header, err := reader.Read()
    if err != nil {
        inpf.Close()
        return nil, err
    }
    // check columns (names and num.) against expected in `schema`
    if err := cr.checkColumns(header); err != nil {
        inpf.Close()
        return nil, err
    }
    // create `out` channel
    outStream := make(chan []string)
    go func() {
        // for graceful exit...
        defer inpf.Close()
        defer close(outStream)
        // do the job
        for {
            row, err := reader.Read()
            if err == io.EOF {
                // end of file
                break
            }
            // read error (inc. I/O operation failure)
            if err != nil {
                // todo: log err
                break
            }
            // save into our stream
            outStream <- row
        }
    }()
    // return stream
    return outStream, nil
}