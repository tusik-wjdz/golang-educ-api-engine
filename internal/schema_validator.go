package internal

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "slices"
    "strconv"
    "strings"
)

const (
    SV_ERRMSG_PARAM_TYPE_MISMATCH string    = "Param `%s` must be: %s."
    SV_ERRMSG_PARAM_UNKNOWN string          = "Unknown param name in request body: `%s`. REFUSED."    
    SV_ERRMSG_PARAM_REQUIRED string         = "Param `%s` is required."
    SV_ERRMSG_URL_ONLY string               = "Param `%s` should be passed as URL param."
    SV_ERRMSG_STR_PARAM_EMPTY string        = "Param `%s` can't be an empty string."
    SV_ERRMSG_CAST_NO_VAL string            = "Unable to cast empty param `%s` w/o def. value."

    SV_HDR_JSON                             = "application/json"
    SV_HDR_FORM_ENCODED                     = "application/x-www-form-urlencoded"
    SV_HDR_FORM_DATA                        = "multipart/form-data"
)

type SchemaValidator struct {
    endpoint 	Route
    eName		string
    ePath 		string
    httpMethod	string
    schema 		[]IncomingParam
}

// constructor
func (SchemaValidator) Build(r *Route, passWithoutSchema bool) *SchemaValidator {
    // try find schema
    s, exists  := ParamList[r.Name]
    if (! exists && ! passWithoutSchema) {
        panic("No schema detected!")
    }

    validator := SchemaValidator{
        endpoint: *r,
        schema: s,
        // wrapper for fast access
        eName: r.Name,
        ePath: r.Path,
        httpMethod: r.Method,
    }

    return &validator
}
// region casts

func (sv *SchemaValidator) tryCastBool(v IncomingValue) (IncomingValue, bool) {
    res, err := strconv.ParseBool(v.GetRawValue())
    if err != nil {
        v.RawValue, v.Boolean  = "false", false        
        return v, false     
    }
    v.Boolean = res
    return v, true
}

func (sv *SchemaValidator) tryCastNumber(v IncomingValue) (IncomingValue, bool) {
    if v.RawValue == "null" || v.RawValue == "nil" {
        // any numeric type based on this will be coverted to nil (pointer)
        v.RawValue, v.Number = "", 0
        // acceptable, so... return true
        return v, true
    }
    res, err := strconv.ParseFloat(v.RawValue, 64)
    if err != nil {
        v.RawValue, v.Number  = "", 0 // just for protection
        return v, false
    }
    v.Number = res
    return v, true
}

func (sv *SchemaValidator) tryCastString(v IncomingValue) (IncomingValue, bool) {
    // todo add check and sanitizer
    v.Text = v.RawValue
    return v, true
}

func (sv *SchemaValidator) tryCastIncomingParam(paramInfo *IncomingParam, v IncomingValue) (IncomingValue, bool) {    
    switch (paramInfo.Type) {
        case PARAM_TYPE_BOOL:
            return sv.tryCastBool(v)
        case PARAM_TYPE_STRING:
            return sv.tryCastString(v)
        case PARAM_TYPE_NUMBER:
            return sv.tryCastNumber(v)
        default:
            return v, false
    }
}
// endregion

// adds URL params to payload
func (sv *SchemaValidator) appendURLParams(payload map[string]IncomingValue, urlParams *[]string, r *http.Request) {
    // grab everything passed via URL Query
    for k, values := range r.URL.Query() { // store all in dedicated temp. array to check URLOnly attribute
        if len(values) > 0 {
            // 1) check
            _, exists := payload[k]            
            if exists {
                // looks like added via PUT / POST    
                // consider log
            }
            payload[k] = IncomingValue{
                Exists:     true,
                RawValue:   values[0],
            }
            *urlParams = append(*urlParams, k)
        }
    }
}
// main validate method (against specified schema in conf.)
func (sv *SchemaValidator) Validate() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
            var payload map[string]IncomingValue    = make(map[string]IncomingValue)
            var urlParams []string                  = make([]string, 16)
            // or POST / PUT
            if sv.httpMethod == http.MethodPost || sv.httpMethod == http.MethodPut {
                // fetch header
                cType := r.Header.Get("Content-Type")
                if r.ContentLength == 0 || cType == "" {
                    goto C_ANYWAY // continue anyway (on empty form)
                }
                if strings.Contains(cType, SV_HDR_JSON) {
                    bodyBytes, err := io.ReadAll(r.Body)
                    if err != nil {
                        // return error. 
                        http.Error(rw, "Unable to read request body.", http.StatusInternalServerError)
                        return
                    }
                    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

                    if len(bodyBytes) > 0 {
                        err = json.Unmarshal(bodyBytes, &payload)
                        if err != nil {
                            http.Error(rw, "Invalid JSON structure!", http.StatusBadRequest)
                            return
                        }
                    }
                } else if strings.Contains(cType, SV_HDR_FORM_ENCODED) || strings.Contains(cType, SV_HDR_FORM_DATA) {
                    r.ParseMultipartForm(32 << 22)
                    for k, values := range r.PostForm {
                        if len(values) > 0 {
                            payload[k] = IncomingValue{Exists: true, RawValue: values[0]};
                        }
                    }
                } else {
                    http.Error(
                        rw, "Unsupported data format.", http.StatusUnsupportedMediaType,
                    )
                    return
                }
            }
            // GET / PUT / DELETE (todo:)
            if sv.httpMethod == http.MethodGet || sv.httpMethod == http.MethodPut || sv.httpMethod == http.MethodDelete {
                // MUST be after PUT/POST collectors
                sv.appendURLParams(payload, &urlParams, r)
            }
C_ANYWAY:
            // time to check against schema
            if ! sv.checkAgainstSchema(payload, &urlParams, rw) {
                // failed
                return
            }
            ctx := context.WithValue(r.Context(), validatedDataKey, payload)
            next.ServeHTTP(rw, r.WithContext(ctx))
        })
    }
}
// checks incoming values against schema rules
func (sv *SchemaValidator) checkAgainstSchema(
    payload map[string]IncomingValue,
    urlParams *[]string,
    rw http.ResponseWriter,
) bool {
    // init schema map
    schemaMap := make(map[string]IncomingParam)
    // check (with type casting)
    for _, paramInfo := range sv.schema {
        schemaMap[paramInfo.Name] = paramInfo
        v, hasKey := payload[paramInfo.Name]
        // check if param exists
        if (paramInfo.IsRequired && !hasKey) {
            http.Error(rw, fmt.Sprintf(SV_ERRMSG_PARAM_REQUIRED, paramInfo.Name), http.StatusBadRequest)            
            return false
        }
        // we need it in `wider`` scope
        var ok bool
        resVal := IncomingValue{}
        if (hasKey) {
            if (paramInfo.UrlOnly && ! slices.Contains(*urlParams, paramInfo.Name)) {
                http.Error(
                    rw, fmt.Sprintf(SV_ERRMSG_URL_ONLY, paramInfo.Name), http.StatusBadRequest,
                )
                return false
            }            
            // now we can continue
            resVal, ok      = sv.tryCastIncomingParam(&paramInfo, v)
            if !ok {
                errMsg := fmt.Sprintf(SV_ERRMSG_PARAM_TYPE_MISMATCH, paramInfo.Name, paramInfo.Type)
                http.Error(rw, errMsg, http.StatusBadRequest)
                return false
            }
        } else {
            if paramInfo.Default == "" {
                // put empty string
                resVal = IncomingValue{
                    Exists: false,
                    RawValue: "",
                }
            } else {
                // Exists (because of def. value) but without `raw` value
                iV := IncomingValue{
                    Exists: true,
                    RawValue: "",
                } // just for transparency
                iV.AddUnknownType(paramInfo.Default)
                resVal = iV
            }
        }
        // one more check
        if paramInfo.Type == PARAM_TYPE_STRING && resVal.RawValue == "" {
            if ! paramInfo.CanBeEmpty {
                // well...
                http.Error(rw, fmt.Sprintf(SV_ERRMSG_STR_PARAM_EMPTY, paramInfo.Name), http.StatusBadRequest)
            }
        }
        payload[paramInfo.Name] = resVal
    }
    // small protection -> check for injected extra-fields (just for sure)
    for reqField := range payload {
        _, inSchema := schemaMap[reqField]
        if !inSchema {
            http.Error(
                rw, fmt.Sprintf(SV_ERRMSG_PARAM_UNKNOWN, reqField), http.StatusBadRequest,
            )
            return false
        }
    }
    return true
}