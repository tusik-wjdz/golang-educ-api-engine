package internal

import "strconv"

// small helper to avoid casting issues (especially NIL casting)
// dedicated for SchemaValidator and handlers

type IncomingValue struct {
    Exists		bool
    RawValue 	string
    Number		float64
    Text		string
    Boolean		bool
    TypeOf		string	
}

func (iv IncomingValue) GetRawValue() string {
    return iv.RawValue
}

func (iv IncomingValue) GetText() string {
    return iv.Text
}

func (iv IncomingValue) GetBool() bool {
    return iv.Boolean
}

func (iv IncomingValue) GetNumberAsString() string {
    var result string
    // parse check
    _ , err := strconv.ParseFloat(iv.GetRawValue(), 64) // try cast to float64 ... just in case
    if err != nil || iv.RawValue == "" {
        result = "NULL"
    } else {
        result = iv.RawValue
    }	
    return result
}


func FetchNumAsPtr[PT Number](iv IncomingValue) *PT {
    if !iv.Exists || iv.RawValue == "" {
        return nil
    }
    return ParseNumber[PT](iv.RawValue)
}

func FetchNum[PT Number](iv IncomingValue) PT {
    if !iv.Exists || iv.RawValue == "" {
        return PT(0)
    }
    r := ParseNumber[PT](iv.RawValue)
    if r == nil {
        return PT(0)
    }
    return *r
}

// TODO: is it working?
func FetchNumOutPtr[PT Number](iv IncomingValue, ptr *PT) PT {
    ptr = nil
    if !iv.Exists || iv.RawValue == "" {
        return PT(0)
    }
    r := ParseNumber[PT](iv.RawValue)
    if r == nil {
        return PT(0)
    }
    ptr = r
    return *r
}

// number getters
// general
func (iv IncomingValue) GetInt() int {	
    return dPtrRefNumberOf(ParseNumber[int](iv.RawValue))
}

func (iv IncomingValue) GetFloat32() float32 {	
    return dPtrRefNumberOf(ParseNumber[float32](iv.RawValue))
}

func (iv IncomingValue) GetFloat64() float64 {	
    return dPtrRefNumberOf(ParseNumber[float64](iv.RawValue))
}

// others
func (iv IncomingValue) GetInt8() int8 {
    res := ParseNumber[int8](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetInt16() int16 {
    res := ParseNumber[int16](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetInt32() int32 {
    res := ParseNumber[int32](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetInt64() int64 {
    res := ParseNumber[int64](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetUint8() uint8 {
    res := ParseNumber[uint8](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetUint16() uint16 {
    res := ParseNumber[uint16](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetUint32() uint32 {
    res := ParseNumber[uint32](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetUint64() uint64 {
    res := ParseNumber[uint64](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func (iv IncomingValue) GetUint() uint {
    res := ParseNumber[uint](iv.RawValue)	
    return dPtrRefNumberOf(res)
}

func dPtrRefNumberOf[PT Number](v *PT) PT {
    if (v == nil) {
        return 0
    }
    return *v
}

// converts to easy in manipulate (for API) types
func (iv *IncomingValue) AddUnknownType(t any) {	
    switch v := t.(type) {
    case int:
        iv.Number 	= float64(v)
        iv.TypeOf 	= "int"
    case int64:
        iv.Number 	= float64(v)
        iv.TypeOf 	= "int64"
    case bool:
        iv.Boolean 	= v
        iv.TypeOf 	= "bool"
    case float64:
        iv.Number 	= v
        iv.TypeOf 	= "float64"
    case string:
        iv.Text 	= v
        iv.TypeOf	= "string"
    default:
        return
    }
}

// parses string and trying to return number specified as GENERIC (PT)
// if string value is NaN or can't be converted to expected type, nil will be returned instead
// use with caution
func ParseNumber[PT Number](v string) *PT {
    var initial PT
    var result PT

    switch any(initial).(type) {
        // ints
        case int, int8, int16, int32, int64:
            r, err := strconv.ParseInt(v, 10, 64)
            if err != nil {
                // consider log
                return nil
            }
            result = PT(r)
        // unsigned ints
        case uint, uint8, uint16, uint32, uint64:
            r, err := strconv.ParseUint(v, 10, 64)
            if err != nil {
                return nil
            }
            result = PT(r)
        // floats (single, double)
        case float32, float64:
            r, err := strconv.ParseFloat(v, 64)
            if err != nil {
                return nil
            }
            result = PT(r)

        default:
            // unknown type / or NaN -> better to return nil
            return nil
    }
    return &result
}

func (iv *IncomingValue) Convert(t string, v string) any {	
    switch t {
    case "int":
        if res, err := strconv.ParseInt(v, 10, 32); err != nil {
            return 0
        } else {
            return int(res)
        }
    case "int64":
        if res, err := strconv.ParseInt(v, 10, 64); err != nil {
            return 0
        } else {
            return int(res)
        }
    case "bool":
        if res, err := strconv.ParseBool(v); err != nil {
            return false
        } else {
            return res
        }
    case "float64":
        if res, err := strconv.ParseFloat(v, 64); err != nil {
            return false
        } else {
            return res
        }
    }
    return nil
}

