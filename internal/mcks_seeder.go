package internal

import (
	// db "test-api/internal/db"
	//"strconv"
	//"fmt"
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	dbc "test-api/internal/pkg/psqlservice"
	"test-api/internal/pkg/tools"
	"time"
)

// region wheater_stations
//
// ======================
// WEATHER STATIONS SEEDER
// ======================

const CSV_PATH_WEATHER_STATIONS = "./static/weather_stations_seed.csv"
const CSV_PATH_PRODUCTS 		= "./static/random_data.csv"

var wsCsvRequiredColumns = []string{
	"city",
	"temperature",
	"humidity",
	"wind",
	"wind gust (1h)",
	"sig_weather",
	"precipitation (1h)",
	"pressure (hPa)",
	"low_clouds coverage (%)",
	"medium_clouds coverage (%)",
	"high_clouds coverage (%)",
}

func SeedWeatherStationsFromStaticCsv() {
	pool 		:= GetEnv().GetConnPool()
	wStations 	:= NewWeatherStationRepository(dbc.GetPsqlService[WeatherStation](pool))
	cr 				:= tools.GetReader()
	// we need pattern (should be defined above this ;) )
	cr.ReqColumns 	= wsCsvRequiredColumns
	stream, err 	:= cr.ReadCsv(CSV_PATH_WEATHER_STATIONS)
	if err != nil {
		log.Fatalf("Error while trying to read from CSV: %v", err)
		return
	}
	// ws := WeatherStation{}
	for row := range stream {		
		city 			:= row[0]
		temp, err 		:= ParseFloat32(row[1])
		humidity, err	:= ParseUint8(row[2])
		wind, err		:= ParseFloat32(row[3])
		gust, err		:= ParseFloat32(row[4])
		sigw			:= row[5]
		prec, err		:= ParseFloat32(row[6])
		pressure, err	:= ParseUint16(row[7])
		lCloudsCov, err := ParseUint8(row[8])
		mCloudsCov, err := ParseUint8(row[9])
		hCloudsCov, err := ParseUint8(row[10])
		
		// todo: log each one but for now last is ok
		if (err != nil) {
			log.Fatalf("Unable to cast value from CSV : %v", err)
		}
		// assign
		ws := WeatherStation{
			City: 				city,
			Temperature: 		&temp,
			Humidity: 			&humidity,
			Wind: 				&wind,
			WindGust: 			&gust,
			SigWeather: 		sigw,
			Precipitation: 		&prec,
			Pressure: 			&pressure,
			LowCloudsCoverage: 	&lCloudsCov,
			MidCloudsCoverage:	&mCloudsCov,
			HighCloudsCoverage: &hCloudsCov,
		}		
		// fmt.Println(ws)
		// time.Sleep(1 * time.Second)
		wStations.Persist(&ws)
	}
	fmt.Println("Time to flush data...")	
	time.Sleep(3 * time.Second)	
	fmt.Println("Start.")
	saveErr := wStations.Flush(context.Background())
	if saveErr != nil {
		fmt.Printf("Save failed. Something went wrong: %v", saveErr)
		return
	}
	fmt.Println("Looks fine. :)")	
}

// endregion

// region products

var prodCsvRequiredColumns = []string{
	"name", "price", "qty", "description", "color",
}

func SeedProductsFromStaticCsv() {
	// hardcoded ;p
	uIds := []int{70, 71, 72}
	// end of hardcoded ;)
	ps := GetProductService() 
	cr 			:= tools.GetReader()
	// we need pattern (should be defined above this ;) )
	cr.ReqColumns 	= prodCsvRequiredColumns
	stream, err 	:= cr.ReadCsv(CSV_PATH_PRODUCTS)
	if err != nil {
		log.Fatalf("Error while trying to read from CSV: %v", err)
		return
	}
	for row := range stream {
		name			:= row[0]
		price, err 		:= ps.pConvertToCents(row[1])
		if err != nil {
			continue
		}				
		qtyInt32, err 	:= ParseInt32(row[2])
		if err != nil {
			continue
		}
		qty				:= int(qtyInt32)
		description		:= row[3]
		color			:= row[4]		
		// assign
		p := Product{
			Name: 			name,
			Price: 			price,
			Qty: 			qty,
			Description: 	description,
			Color: 			color,
			CreatedAt: 		time.Now().Unix(),
			CreatedBy:		uIds[rand.IntN(2)],
			// todo CreatedBy
		}
		ps.Products.Persist(&p)		
	}
	fmt.Println("Time to flush data...")
	time.Sleep(3 * time.Second)	
	fmt.Println("Start.")
	saveErr := ps.Products.Flush(context.Background())
	if saveErr != nil {
		fmt.Printf("Save failed. Something went wrong: %v", saveErr)
		return
	}
	fmt.Println("Looks fine. :)")
}

// endregion

