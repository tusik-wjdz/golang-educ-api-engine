package internal

import (
	"context"
	"log"
	"test-api/internal/db"
	dbc "test-api/internal/pkg/psqlservice"
	"time"
)

type WeatherStationService struct {
    Stations 		*db.Repository[*WeatherStation]
}

// Service constructor
func GetWeatherStationService() *WeatherStationService {
    return &WeatherStationService{
        Stations: NewWeatherStationRepository(dbc.GetPsqlService[WeatherStation](GetEnv().GetConnPool())),
    }
}

// WS by `id`
func (wss *WeatherStationService) Find(ctx context.Context, id int) (*WeatherStation, bool) {
    ws, ok := wss.Stations.Find(ctx, id)
    if ! ok {
        return ws, false
    }
    return ws, true
}
// list WS
func (wss *WeatherStationService) List(
    ctx context.Context,
    orderBy string,
    direction string,
    limit int,
    offset int,
) ([]*WeatherStation, error) {
    uList, err := wss.Stations.FindAll(ctx, limit, offset, orderBy, direction)
    if err != nil {
        log.Println(err)
        return nil, ErrorFactory(ERR_GET_LIST, "WeatherStation")
    }
    return uList, nil
}
// create WeatherStation entity
func (wss *WeatherStationService) Create(
    ctx context.Context,
    city string,
    temp *float32,
    humidity *uint8,
    wind *float32,
    gust *float32,
    sigW string,
    prec *float32,
    pressure *uint16,
    lCloudsCov *uint8,
    mCloudsCov *uint8,
    hCloudsCov *uint8,
) (*WeatherStation, error) {
    // pre-validation
    if ! wss.validateTemp(temp) {
        return nil, ErrorFactory(ERR_TEMP_OUT_OF_RANGE)
    }
    if ! wss.validateWind(wind) || ! wss.validateWind(gust) {        
        return nil, ErrorFactory(ERR_WIND_OUT_OF_RANGE)
    }
    if ! wss.validateHumidity(humidity) {
        return nil, ErrorFactory(ERR_HUMIDITY_OUT_OF_RANGE)
    }
    if ok, _ := wss.validateCloudsCoverage(lCloudsCov, mCloudsCov, hCloudsCov); ! ok {
        return nil, ErrorFactory(ERR_CLDS_COVERAGE_OUT_OF_RANGE)
    }
    // time to create entity
    cTime := time.Now().Unix()
    ws := WeatherStation{
        City:               city,
        Temperature:        temp,
        Humidity:           humidity,
        Wind:               wind,
        WindGust:           gust,
        SigWeather:         sigW,
        Precipitation:      prec,
        Pressure:           pressure,
        LowCloudsCoverage:  lCloudsCov,
        MidCloudsCoverage:  mCloudsCov,
        HighCloudsCoverage: hCloudsCov,
        CreatedAt:          cTime,
        UpdatedAt:          &cTime,
    }
    // save into repository
    err := wss.Stations.Save(ctx, &ws)
    if err != nil {
        return &WeatherStation{}, ErrorFactory(ERR_ENTITY_SAVE, "WeatherStation")
    }
    return &ws, nil
}
// update (if changed)
func (wss *WeatherStationService) Update(
    ctx context.Context,
    ws *WeatherStation,
    incomingData map[string]IncomingValue,
) (*WeatherStation, error) {
    if ws.ID < 1 {
        return ws, ErrorFactory(ERR_ENTITY_UNSAVED, "WeatherStation")
    }
    isModified := false    
    // passed params
    for k, iV := range incomingData {
        switch k {
        case "temperature":
            if ApplyNumVPtr(&ws.Temperature, iV) { isModified = true }
        case "humidity":
            if ApplyNumVPtr(&ws.Humidity, iV) { isModified = true }
        case "wind":
            if ApplyNumVPtr(&ws.Wind, iV) { isModified = true }
        case "wind_gust":
            if ApplyNumVPtr(&ws.WindGust, iV) { isModified = true }
        case "precipitation":
            if ApplyNumVPtr(&ws.Precipitation, iV) { isModified = true }
        case "pressure":
            if ApplyNumVPtr(&ws.Pressure, iV) { isModified = true }
        case "low_clouds_coverage":
            if ApplyNumVPtr(&ws.LowCloudsCoverage, iV) { isModified = true }
        case "mid_clouds_coverage":
            if ApplyNumVPtr(&ws.MidCloudsCoverage, iV) { isModified = true }
        case "high_clouds_coverage":
            if ApplyNumVPtr(&ws.HighCloudsCoverage, iV) { isModified = true }
        case "sig_weather":
            if (! iV.Exists) {
                continue
            }
            sigW := iV.GetText()
            if (ws.SigWeather == sigW) {
                continue
            }
            // otherwise
            ws.SigWeather, isModified = sigW, true
        }
    }

    if isModified {
        // post-validation
        if ! wss.validateTemp(ws.Temperature) {
            return nil, ErrorFactory(ERR_TEMP_OUT_OF_RANGE)
        }
        if ! wss.validateWind(ws.Wind) || ! wss.validateWind(ws.WindGust) {        
            return nil, ErrorFactory(ERR_WIND_OUT_OF_RANGE)
        }
        if ! wss.validateHumidity(ws.Humidity) {
            return nil, ErrorFactory(ERR_HUMIDITY_OUT_OF_RANGE)
        }
        if ok, _ := wss.validateCloudsCoverage(
            ws.LowCloudsCoverage,
            ws.MidCloudsCoverage,
            ws.HighCloudsCoverage,
        ); ! ok {
            return nil, ErrorFactory(ERR_CLDS_COVERAGE_OUT_OF_RANGE)
        }
        // set UpdatedAt
        uTime           := time.Now().Unix()
        ws.UpdatedAt    = &uTime
        // persist due to strategy then save
        err := wss.Stations.Persist(ws).Flush(ctx)
        if err != nil {
            log.Printf("Couldn't update this user %v. \n", ws.ID)
            return ws, ErrorFactory(ERR_ENTITY_UDPATE, "WeatherStation")
        }
    }
    return ws, nil
}
// remove by entity
func (wss *WeatherStationService) Remove(ctx context.Context, ws *WeatherStation) error {
    if ws.ID < 1 {
        return ErrorFactory(ERR_ENTITY_UNSAVED, "WeatherStation")
    }
    // "redundant" persist
    affRows, err := wss.Stations.Persist(ws).Delete(ctx)
    if err != nil {
        // log "base" error
        log.Println(err)
        return ErrorFactory(ERR_ENTITY_REMOVE, "WeatherStation")
    }
    if affRows != 1 {
        log.Printf("Invalid removed rows:%d\n", affRows)
        // consider panic / general error ;)
    }
    // looks ok    
    return nil
}
// remove by id
func (wss *WeatherStationService) RemoveById(ctx context.Context, id int) error {
    if ws, ok := wss.Find(ctx, id); ok {
        return wss.Remove(ctx, ws)
    }
    return ErrorFactory(ERR_ENTITY_NOT_FOUND, "WeatherStation")
}

// helpers
// checks temp. range
func (wss *WeatherStationService) validateTemp(v *float32) bool {
    if nil == v { return true } // ignore
    return NumInRange(-273.15, 1000, *v)
}
// wind range validation 
func (wss *WeatherStationService) validateWind(v *float32) bool {
    if nil == v { return true } // ignore
    return NumInRange(0, 1000, *v)
}
// cloud coverage validation
func (wss *WeatherStationService) validateCloudsCoverage(vals ...*uint8) (bool, int) {
    // consider change to map type and specify witch one is incorrect
    for i, v := range vals {
        if nil == v { continue } // ignore
        if ! ValidatePercentageValue(*v) {
            return false, i
        }
    }
    // all ok
    return true, len(vals)
}
// humidity percentage range check
func (wss *WeatherStationService) validateHumidity(v *uint8) bool {
    // consider change to floats
    if nil == v { return true } // ignore
    return NumInRange(1, 100, *v)
}