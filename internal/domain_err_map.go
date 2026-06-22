package internal

// domain error list
const (
    ERR_DM_TEST uint16											= iota + 0xFA00 // important
    ERR_TEMP_OUT_OF_RANGE uint16								= iota + 0xFA00
    ERR_WIND_OUT_OF_RANGE uint16								= iota + 0xFA00
    ERR_HUMIDITY_OUT_OF_RANGE uint16							= iota + 0xFA00
    ERR_CLDS_COVERAGE_OUT_OF_RANGE uint16						= iota + 0xFA00

    ERR_CONV_TOCENTS uint16										= iota + 0xFA00
    ERR_INV_PROD_OWNER uint16									= iota + 0xFA00
    ERR_BULK_MODE_SAVE_FAILED uint16							= iota + 0xFA00
)

// todo: this should be load from a file by lang code
var domainErrorsDict = map[uint16][]string{
    // TEST
    ERR_DM_TEST: 						{"This error test message.", 								"U001", "401"},
    // WEATHER STATION
    ERR_TEMP_OUT_OF_RANGE: 				{"Temperature value is out of range. Check input data.", 	"WS01", "401"},
    ERR_WIND_OUT_OF_RANGE:				{"Wind value is out of range. Check input data.",			"WS02", "401"},
    ERR_HUMIDITY_OUT_OF_RANGE:			{"Humidity value is out of range.. Check input data.", 		"WS03", "401"},
    ERR_CLDS_COVERAGE_OUT_OF_RANGE:		{"C. coverage (low, mind or high) out of range.", 			"WS04", "401"},
    // PRODUCT
    ERR_CONV_TOCENTS:					{"Convert to cents failed. Reason: [%s]",					"PR10", "401"},
    ERR_INV_PROD_OWNER:					{"You are not the owner of this product.",					"PR11", "401"},
    ERR_BULK_MODE_SAVE_FAILED:			{"Bulk mode import / save entities failed! Check logs.",	"PR12", "401"},
}

