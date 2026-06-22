package internal

import (
    "maps"
    "fmt"
    "net/http"
    "strconv"
)

// treat is as const
var errDictionary map[uint16][]string

// Factory
func ErrorFactory(id uint16, params... any) *ApiResponseError {
    // consider not use ANY interface
    ApiErr          := &ApiResponseError{}
    msg, iCode, httpCode := ApiErr.getErrInfo(id)
    // set message
    ApiErr.Msg      = fmt.Sprintf(msg, params...)
    // set codes (http and API internal)
    ApiErr.HttpCode = httpCode
    ApiErr.ICode    = iCode
    return ApiErr
}
// General api response error
type ApiResponseError struct {
    HttpCode        int
    Msg             string
    ICode           string
    IAlias          string
    Err             error
}
// gets error info in API's required format with message, internal code and http code
func (e ApiResponseError) getErrInfo(id uint16) (message string, iCode string, httpCode int) {
    if len(errDictionary) == 0 {
        // main errors
        errDictionary = mainErrorsDict
        // merge with domain errors dict
        maps.Copy(errDictionary, domainErrorsDict)        
    }
    result, exists := errDictionary[id]
    if !exists {
        result = errDictionary[ERR_UNKNOWN]
    }
    // prepare results
    message, iCode 	= result[0], result[1]
    httpCode, _ 	= strconv.Atoi(result[2])
    return message, iCode, httpCode
}
// implement Error interface
func (e *ApiResponseError) Error() string {
    if e.Err != nil {
        e.HttpCode = http.StatusInternalServerError
        e.Msg = "Oops..."
        // log internal err        
        return fmt.Errorf("Code: %d, Message: %s", e.HttpCode, e.Msg).Error()
    }
    return fmt.Sprintf("Code: %d, Message: %s, ICode: %s", e.HttpCode, e.Msg, e.ICode)
}

func (e *ApiResponseError) Unwrap() error {
    return e.Err
}

func (e *ApiResponseError) getErrStrf() string {
    return fmt.Sprintf("Code: %d, Message: %s, ICode: %s", e.HttpCode, e.Msg, e.ICode)
}

func (e *ApiResponseError) Message(m string) *ApiResponseError {
    e.Msg = m
    return e
}