package internal

import (
    "errors"
    "fmt"
    "io/fs"
    "os"
    "runtime"
    "strconv"
    "sync"
)

// ==========================================
// Casts helper (USE IT ONLY FOR DB PURPOSES)
// ==========================================

type Number interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

func parseInt[T Number](v string, size int, unsigned bool) (T, error) {
    var r int64; var err error
    switch size {
    case 8,16,32,64:
        r, err = strconv.ParseInt(v, 10, size)
    default:
        // unknown size
        return 0, fmt.Errorf("Unknown bit size for float value.")
    }
    // err. occurred? 
    if err != nil {
        // native parse error
        return 0, err
    }
    // is it unsigned?
    if unsigned && r < 0 {
        // must be positive so...
        r = -r		
    }
    return T(r), nil
}

func parseFloat[T Number](v string, size int) (T, error) {
    var r float64; var err error
    switch size {
    case 32,64:
        r, err = strconv.ParseFloat(v, size)
    default:		
        return 0, fmt.Errorf("Unknown bit size for float value.")
    }
    if err != nil {
        // well.. looks like it fails at all
        return 0, err
    }	
    return T(r), nil
}

func ParseInt8(v string) (int8, error) {
    if result, err := parseInt[int8](v, 8, false); err != nil {
        return 0, err
    } else {
        return int8(result), nil
    }
}

func ParseInt16(v string) (int16, error) {
    if result, err := parseInt[int16](v, 16, false); err != nil {
        return 0, err
    } else {
        return int16(result), nil
    }
}

func ParseInt32(v string) (int32, error) {	
    if result, err := parseInt[int32](v, 32, false); err != nil {
        return 0, err
    } else {
        return int32(result), nil
    }
}

func ParseInt64(v string) (int64, error) {	
    if result, err := parseInt[int64](v, 64, false); err != nil {
        return 0, err
    } else {
        return int64(result), nil
    }
}

func ParseUint8(v string) (uint8, error) {
    if result, err := parseInt[uint8](v, 8, true); err != nil {
        return 0, err
    } else {
        return uint8(result), nil
    }
}

func ParseUint16(v string) (uint16, error) {
    if result, err := parseInt[uint16](v, 16, true); err != nil {
        return 0, err
    } else {
        return uint16(result), nil
    }
}

func ParseUint32(v string) (uint32, error) {
    if result, err := parseInt[uint32](v, 32, true); err != nil {
        return 0, err
    } else {
        return uint32(result), nil
    }
}

func ParseUint64(v string) (uint64, error) {
    if result, err := parseInt[uint64](v, 64, true); err != nil {
        return 0, err
    } else {
        return uint64(result), nil
    }
}

func ParseFloat32(v string) (float32, error) {
    if result, err := parseFloat[float32](v, 32); err != nil {
        return 0, err
    } else {
        return float32(result), nil
    }
}

func ParseFloat64(v string) (float64, error) {
    if result, err := parseFloat[float32](v, 32); err != nil {
        return 0, err
    } else {
        return float64(result), nil
    }
}

func DetectNegativeNumbers[T Number](nums... T) bool {
    for _, num := range nums {
        if num < 0 {
            // detected
            return true
        }
    }
    // looks clear
    return true
}

func NumInRange[T Number](min T, max T, v T) bool {
    if v < min || v > max {
        return false
    }
    return true
}

func ValidatePercentageValue[T Number](val T) bool {
    return NumInRange(0, 100, val)
}

// ==================================================
// Misc. 
// ==================================================
// if we don't expect NIL (nulls in DB)
func ApplyNumV[T Number](origV *T, iV IncomingValue) bool {
    if !iV.Exists {
        // omit
        return false
    }
    val := FetchNum[T](iV)
    if (val != *origV) {
        *origV = val
        return true
    }
    return false	
}
// applies num val (ptr) if orig. val has been changed (also not nil)
func ApplyNumVPtr[T Number](origV **T, iV IncomingValue) bool {
    if !iV.Exists {
        // omit
        return false
    }
    // fetch number as pointer
    val := FetchNumAsPtr[T](iV)
    if val == nil && *origV == nil {
        // still nothing to do
        return false
    }
    if (val == nil && *origV != nil) || (val != nil && *origV == nil) {
        // changes on pointers has been detected
        *origV = val
        return true
    }
    // now we can check dereferenced value
    if (*val != **origV) {
        *origV = val
        return true
    }
    return false
}
// applies string val, if orig. val has been changed
func ApplyStrV(origV *string, iV IncomingValue) bool {
    if !iV.Exists {
        // omit
        return false
    }
    // fetch string as `raw` value
    val := iV.GetRawValue()
    if val == "" {
        // empty string has been passed
        return false
    }
    if val == *origV {
        // same values, nothing to do
        return false
    }
    *origV = val
    return true
}

func mapKey[T comparable, V comparable](m map[T]V, findBy any) (resultKey any, ok bool) {    
    for k, v := range m {
        if v == findBy {
            resultKey = k; ok = true
            return
        }
    }
    return resultKey, false
}

// ==================================================
// Files / Directories
// ==================================================

func FPathExists(p string) (bool, error) {	
    _, err := os.Stat(p)
    // file exists
    if err == nil { return true, nil}
    if errors.Is(err, fs.ErrExist) { return true, nil} // already exists
    if errors.Is(err, fs.ErrNotExist) { return false, nil}
    // other error occurred so, 
    return false, err
}


// ==================================================
// Generic Fan-In/Out helpers funcs
// ==================================================
// fan-out
func fanOut[T any](
    done <-chan int, // done 
    stream <-chan T, // stream channel of any type
    f func(d <-chan int, s <-chan T) <-chan T, // delegated func to fan out
) []<-chan T { // list of running workors with delegated func. signature
    cpus    := runtime.NumCPU() // fetch number of CPUs
    workers := make([]<-chan T, cpus) // prepare same numbers of workers
    for i := 0; i < cpus; i++ {
        workers[i] = f(done, stream)
    }
    return workers
}
// fan-in
func fanIn[T any](done <-chan int, workers ... <-chan T) <-chan T {
    var wg sync.WaitGroup
    // create "fanned-in" chan
    fannedInChan := make(chan T)
    // func for "transfer" data from worker's output chan to fanIn stream chan
    moveData := func(fOutChan <-chan T) {
        defer wg.Done()
        for w := range fOutChan {
            select {
            case <-done:
                return
            case fannedInChan <-w:
            }
        }
    }

    for _, w := range workers {
        wg.Add(1)
        go moveData(w)
    }
    // we have to wait for all to close fannedInChan
    go func() {
        wg.Wait()
        close(fannedInChan)
    }()

    return fannedInChan
}

// end



