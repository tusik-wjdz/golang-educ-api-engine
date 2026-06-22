package internal

import (
    db "test-api/internal/db"
    dbc "test-api/internal/pkg/psqlservice"
)


// ===========================
// WEATHER STATIONS REPOSITORY
// ===========================

func (ws *WeatherStation) SetID(id int) {	
    ws.ID = id
}

func (ws *WeatherStation) GetStruct() any {
    return ws
}

func (ws *WeatherStation) GetID() int {
    return ws.ID
}

func (ws *WeatherStation) GetTableName() string {
    return "public.dm_weather_stations"
}

func (ws *WeatherStation) GetColumns() []string {
    return []string{
        "city",
        "temperature",
        "humidity",
        "wind",
        "wind_gust",
        "sigw",
        "precipitation",
        "pressure",
        "low_clouds_coverage",
        "mid_clouds_coverage",
        "high_clouds_coverage",
        "created_at",
        "updated_at",
    }
}

func (ws *WeatherStation) GetValues() []any {
    return []any{
        ws.City,
        ws.Temperature,
        ws.Humidity,
        ws.Wind,
        ws.WindGust,
        ws.SigWeather,
        ws.Precipitation,
        ws.Pressure,
        ws.LowCloudsCoverage,
        ws.MidCloudsCoverage,
        ws.HighCloudsCoverage,
        ws.CreatedAt,
        ws.UpdatedAt,
    }
}

func (ws *WeatherStation) GetKeyValuePair() map[string]any {
    // still not working	
    return map[string]any {}
}

func NewWeatherStationRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver)
}

// ===========================
// PRODUCTS REPOSITORY (EXAMPLE)
// ===========================

func (p *Product) SetID(id int) {	
    p.ID = id
}

func (p *Product) GetStruct() any {
    return p
}

func (p *Product) GetID() int {
    return p.ID
}

func (p *Product) GetTableName() string {
    return "public.dm_products"
}

func (p *Product) GetColumns() []string {
    return []string{
        "name",
        "price",
        "qty",
        "description",
        "color",
        "checksum",
        "created_at",
        "updated_at",
        "created_by",
        "updated_by",
    }
}

func (p *Product) GetValues() []any {
    return []any{
        p.Name,
        p.Price,
        p.Qty,
        p.Description,
        p.Color,
        p.Checksum,
        p.CreatedAt,
        p.UpdatedAt,
        p.CreatedBy,
        p.UpdatedBy,
    }
}

func (p *Product) GetKeyValuePair() map[string]any {
    // todo:
    return map[string]any {}
}
// constructor
func NewProductRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver).SetAllowedCols(
        map[string]string{
            "n":            "name",
            "p":            "price",
            "q":            "qty",
            "col":          "color",
            "fingerprint":  "checksum",
            "created_by":   "created_by",
            "created_at":   "created_at",
    })
}