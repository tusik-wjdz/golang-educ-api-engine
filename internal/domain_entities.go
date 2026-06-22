package internal

type (
    WeatherStation struct {
        // pointers as protection "against NULLables" :)
        ID          		int      	`db:"id"`
        City 				string		`db:"city"`
        Temperature			*float32	`db:"temperature"`
        Humidity			*uint8		`db:"humidity"`
        Wind				*float32	`db:"wind"`
        WindGust			*float32	`db:"wind_gust"`
        SigWeather			string		`db:"sigw"`
        Precipitation		*float32	`db:"precipitation"`
        Pressure			*uint16		`db:"pressure"`
        LowCloudsCoverage	*uint8		`db:"low_clouds_coverage"`
        MidCloudsCoverage	*uint8		`db:"mid_clouds_coverage"`
        HighCloudsCoverage	*uint8		`db:"high_clouds_coverage"`
        CreatedAt   		int64       `db:"created_at"`
        UpdatedAt   		*int64		`db:"updated_at"`
    }

    Product struct {
        ID					int			`db:"id"`
        Name				string		`db:"name"`
        Price				int64		`db:"price"`
        Qty					int			`db:"qty"`
        Description			string		`db:"description"`
        Color				string		`db:"color"`
        Checksum            *string     `db:"checksum"`
        CreatedAt			int64		`db:"created_at"`
        UpdatedAt			*int64		`db:"updated_at"`
        CreatedBy			int			`db:"created_by"`
        UpdatedBy			*int		`db:"updated_by"`
    }
)