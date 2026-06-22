package internal

/**
 * Handlers: few fundamental rules:
 *
 * All handlers MUST return "error"
 * All handlers MUST use Respond / ErrorFactory methods
 * All errors SHOULD be prepared in services and
 * ... also ALL ERRORS MUST contains NON-SENSITIVE informations
 * e.g. "User not found.", "Can't read data. Try later..." or something like that.
 * All sensitive data MUST be logged in services or OPTIONALLY 
 * may be use in response but ONLY IN DEBUG MODE!
 *
 * Well, thats it.
 */

import "net/http"

//==============================================
// Weather Stations Handlers
//==============================================

// Route: ws/Find
func FindWeatherStation(rw http.ResponseWriter, r *http.Request) error {
    ctx 			:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    wsService		:= GetWeatherStationService() // grab service
    if ws, ok := wsService.Find(ctx, FetchNum[int](incomingData["id"])); ok { 
        return Respond(rw, r, ws)
    }
    return ErrorFactory(ERR_ENTITY_NOT_FOUND, "WeatherStation")
}

// Route: ws/create
func CreateWeatherStation(rw http.ResponseWriter, r *http.Request) error {
    ctx 			:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)	
    // grab service
    wsService		:= GetWeatherStationService()
    // take data as pointers (only "castable" num. types -> ignore data out of range / not passed / type mismatch etc.)
    wStation, err	:= wsService.Create(
        ctx,
        incomingData["city"].GetText(),
        FetchNumAsPtr[float32](incomingData["temperature"]),
        FetchNumAsPtr[uint8](incomingData["humidity"]),
        FetchNumAsPtr[float32](incomingData["wind"]),
        FetchNumAsPtr[float32](incomingData["wind_gust"]),
        incomingData["sig_weather"].GetText(),
        FetchNumAsPtr[float32](incomingData["precipitation"]),
        FetchNumAsPtr[uint16](incomingData["pressure"]),
        FetchNumAsPtr[uint8](incomingData["low_clouds_coverage"]),
        FetchNumAsPtr[uint8](incomingData["mid_clouds_coverage"]),
        FetchNumAsPtr[uint8](incomingData["high_clouds_coverage"]),
    )	
    if err != nil {
        return err
    }
    result := map[string]any {
        "created_weather_station": wStation,
    }	
    return Respond(rw, r, result)
}

// Route: ws/update
func UpdateWeatherStation(rw http.ResponseWriter, r *http.Request) error {
    ctx 			:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)	
    // grab service
    wsService		:= GetWeatherStationService()
    // grab ID
    id 				:= FetchNum[int](incomingData["id"])
    ws, ok			:= wsService.Find(ctx, id)
    if ! ok {
        // not found
        return ErrorFactory(ERR_ENTITY_NOT_FOUND, "WeatherStation")
    }
    // try update
    ws, err := wsService.Update(ctx, ws, incomingData)
    if err != nil {		
        return err
    }	
    result := map[string]any {
        "updated_weather_station": ws,
    }
    return Respond(rw, r, result)
}

// Route: ws/delete
func DeleteWeatherStation(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    // grab service
    wsService		:= GetWeatherStationService()
    // then entity via passaed ID	
    ws, ok 			:= wsService.Find(ctx, FetchNum[int](incomingData["id"]))
    if ! ok {
        return ErrorFactory(ERR_ENTITY_NOT_FOUND, "WeatherStation")
    }
    err := wsService.Remove(ctx, ws)
    if err != nil {
        return err
    }
    result := map[string]any {
        "deleted_weather_station": ws, // show deleted station
    }
    return Respond(rw, r, result)
}

// Route: ws/list
func ListWeatherStations(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    orderBy, dir    := incomingData["by"].GetText(), incomingData["dir"].GetText()
    list, err 		:= GetWeatherStationService().List(
        ctx,
        orderBy,
        dir,
        FetchNum[int](incomingData["limit"]),
        FetchNum[int](incomingData["offset"]),
    )
    if err == nil {
        return Respond(rw, r, list)
    }
    return err
}

//==============================================
// END: Weather Stations Handlers
//==============================================


//==============================================
// Products Handlers
//==============================================
// Route: products/create
func CreateProduct(rw http.ResponseWriter, r *http.Request) error {
    ctx 			:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)	
    // grab service
    pService		:= GetProductService()
    // set `tradditional approach`
    pService.SetBulkMode(false)
    p, err			:= pService.Create(
        ctx,
        incomingData["name"].GetText(),
        incomingData["price"].GetRawValue(),
        FetchNum[int](incomingData["qty"]),
        incomingData["description"].GetText(),
        incomingData["color"].GetText(),
    )
    if err != nil {
        return err
    }
    return Respond(rw, r, map[string]any {"created_product": p})
}
// Route: products/update
func UpdateProduct(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)	
    // grab service
    pService		:= GetProductService()
    // set `tradditional approach`
    pService.SetBulkMode(false)
    // try find product passed by id
    p, ok := pService.Find(ctx, FetchNum[int](incomingData["id"]))
    if !ok {
        return ErrorFactory(ERR_ENTITY_NOT_FOUND, "product")
    }
    // then update
    p, err := pService.Update(ctx, p, incomingData)
    if err != nil {
        return err
    }
    return Respond(rw, r, map[string]any {"updated_product": p})
}
// Route: products/list
func ListProducts(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    orderBy, dir    := incomingData["by"].GetText(), incomingData["dir"].GetText()
    list, err		:= GetProductService().List(
        ctx,
        orderBy,
        dir,
        FetchNum[int](incomingData["limit"]),
        FetchNum[int](incomingData["offset"]),
    )
    if err != nil {
        return err
    }
    return Respond(rw, r, list)
}
// Route: products/list-by-user
func ListProductsInUserCtx(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    // fetch logged user
    loggedUser, ok	:= GetAuthService().GetLoggedUserPtr(ctx)
    if !ok {
        return ErrorFactory(ERR_CANT_FETCH_LOGGED_USER)
    }
    list, err		:= GetProductService().ListByOwner(
        ctx,
        loggedUser,
        FetchNum[int](incomingData["limit"]),
        FetchNum[int](incomingData["offset"]),
    )
    if err != nil {
        return err		
    }
    return Respond(rw, r, list)
}

// Route: products/delete
func DeleteProduct(rw http.ResponseWriter, r *http.Request) error {
    ctx				:= r.Context()
    incomingData 	:= GetValidatedDataFromContext(ctx)
    ps              := GetProductService()
    p, ok   := ps.Find(ctx, FetchNum[int](incomingData["id"]))
    if ! ok {
        return ErrorFactory(ERR_ENTITY_NOT_FOUND, "Product")
    }    
    err     := ps.Remove(ctx, p)
    if err != nil {
        return err
    }
    result  := map[string]any {
        "deleted_product": p,
    }
    return Respond(rw, r, result)
}

//==============================================
// END: Products handlers
//==============================================
