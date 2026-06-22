package internal

import (
    "bytes"
    "net/http"    
)

/**
 * Response Interceptor
 * for customize Write() and WriteHeader() behavior
 *
 * Author: W.Dzieciol
 **/
type ResponseInterceptor struct {
    http.ResponseWriter  // original ResponseWriter
    StatusCode  int
    BodyBuffer  *bytes.Buffer
}
// ctor
func (ResponseInterceptor) Get(rw http.ResponseWriter) *ResponseInterceptor {
    return &ResponseInterceptor{
        ResponseWriter: rw,
        StatusCode:     http.StatusOK,
        BodyBuffer:     &bytes.Buffer{},
    }
}
// intercepting response data to RAM ... for a while
func(r *ResponseInterceptor) Write(b []byte) (int, error) {
    return r.BodyBuffer.Write(b)
}
// also intercepting status code
func(r *ResponseInterceptor) WriteHeader(statusCode int) {
    r.StatusCode = statusCode
}

