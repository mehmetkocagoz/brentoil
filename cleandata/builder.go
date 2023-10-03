package cleandata

import (
	"fmt"
	"mehmetkocagz/database"
	"mehmetkocagz/datascrape"
	"os"
	"strconv"
	"time"
)

func TableBuilder() {
	database := database.Connect()
	defer database.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS prices (
		timestamp bigint PRIMARY KEY,
		brentoilprice DECIMAL(10, 2) DEFAULT 0.00,
		fuelprice DECIMAL(10, 2) DEFAULT 0.00,
		exchangeRate DECIMAL(10, 2) DEFAULT 0.00
	);
	`
	_, err := database.Exec(createTableQuery)
	if err != nil {
		fmt.Println("Error creating table:", err)
		return
	}

	fmt.Println("Table created successfully.")
}

func FillTableTimestamp() {
	database := database.Connect()
	defer database.Close()

	loc, _ := time.LoadLocation("Europe/Istanbul")
	startDate := time.Date(2018, time.August, 28, 0, 0, 0, 0, loc)
	endDate := time.Date(2023, time.September, 5, 0, 0, 0, 0, loc)

	for startDate.Unix() < endDate.Unix() {
		insertQuery := `
		INSERT INTO prices (timestamp) VALUES ($1)
		`
		_, err := database.Exec(insertQuery, startDate.Unix()*1000)
		if err != nil {
			fmt.Println("Error inserting timestamp:", err)
			return
		}
		startDate = startDate.AddDate(0, 0, 1)
	}
	fmt.Println("Timestamps inserted successfully.")
}

func UpdateTableTimestamp() {
	database := database.Connect()
	defer database.Close()

	row := database.QueryRow("SELECT timestamp FROM prices ORDER BY timestamp DESC LIMIT 1")

	var timestamp int64
	scanError := row.Scan(&timestamp)

	if scanError != nil {
		fmt.Println("UpdateTableTimestamp Error : ", scanError)
	}

	loc, _ := time.LoadLocation("Europe/Istanbul")
	timestampTime := time.Unix(timestamp/1000, 0)
	startDate := timestampTime.Add(24 * time.Hour)
	endDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, loc)

	for startDate.Unix() < endDate.Unix() {
		insertQuery := `
		INSERT INTO prices (timestamp) VALUES ($1)
		`
		_, err := database.Exec(insertQuery, startDate.Unix()*1000)
		if err != nil {
			fmt.Println("Error inserting timestamp:", err)
			return
		}
		startDate = startDate.AddDate(0, 0, 1)
	}
}

func FillTableBrentOilPrice() {
	// Connect to database
	database := database.Connect()
	defer database.Close()

	// Get brentoilprices from bloomberght.com
	brentoilPrices := datascrape.GetBrentOilPrices()

	// Insert brentoilprices to database
	insertQuery := `
		UPDATE prices SET brentoilprice = $1 WHERE timestamp = $2
		`
	for v := range brentoilPrices {
		_, err := database.Exec(insertQuery, brentoilPrices[v].Price, brentoilPrices[v].Timestamp)
		if err != nil {
			fmt.Println("Error inserting brentoilprice:", err)
			return
		}
	}
	// Some of the brentoilprices are missing.
	// So I will fill the missing ones with the previous day's price.
	// Take each row from the database
	rows, err := database.Query("SELECT timestamp from prices ORDER BY timestamp ASC")
	if err != nil {
		fmt.Println("Query has failed: ", err)
	}
	defer rows.Close()
	v := 0
	for rows.Next() {
		var timestamp int64

		scanError := rows.Scan(&timestamp)
		if scanError != nil {
			fmt.Println("Scan has failed: ", scanError)
		}
		if brentoilPrices[v].Timestamp >= timestamp {
			_, err := database.Exec(insertQuery, brentoilPrices[v].Price, timestamp)
			if err != nil {
				fmt.Println("Error inserting brentoilprice:", err)
				return
			}
		} else {
			for brentoilPrices[v].Timestamp < timestamp {
				v++
			}
			_, err := database.Exec(insertQuery, brentoilPrices[v].Price, timestamp)
			if err != nil {
				fmt.Println("Error inserting brentoilprice:", err)
				return
			}
		}
	}

}

func UpdateTableBrentOilPrice() {
	// Connect to database
	database := database.Connect()
	defer database.Close()

	// Get brentoilprices from bloomberght.com
	brentoilPrices := datascrape.GetBrentOilPrices()
	var databaseHolder []datascrape.BrentOilPrice

	rows, err := database.Query("SELECT timestamp,brentoilprice from prices ORDER BY timestamp DESC LIMIT 10")
	defer rows.Close()
	if err != nil {
		fmt.Println("Update table brent oil price error ==> ", err)
	}
	var timestamp int64
	var brentoilprice float64
	for rows.Next() {
		rows.Scan(&timestamp, &brentoilprice)

		dataPoint := datascrape.BrentOilPrice{
			Timestamp: timestamp,
			Price:     brentoilprice,
		}
		databaseHolder = append(databaseHolder, dataPoint)
	}

	for v := len(databaseHolder) - 1; v <= 0; v-- {
		fmt.Println(databaseHolder[v].Price)
		if databaseHolder[v].Price == 0 {
			matchFound := false
			for a := range brentoilPrices {
				fmt.Println(a)
				if brentoilPrices[a].Timestamp == databaseHolder[v].Timestamp {
					matchFound = true
					database.Exec("UPDATE prices SET brentoilprice = $1 WHERE timestamp = $2", brentoilPrices[a].Price, brentoilPrices[a].Timestamp)
					databaseHolder[v].Price = brentoilPrices[a].Price
					break
				}
			}
			if !matchFound {
				database.Exec("UPDATE prices SET brentoilprice = $1 WHERE timestamp = $2", databaseHolder[v+1].Price, databaseHolder[v].Timestamp)
			}
		}
	}
}

func FillTableFuelPrice() {
	// Connect to database
	database := database.Connect()
	defer database.Close()

	// Get fuelprices from tppd.com.tr
	doc := *datascrape.GetFuelPrices()
	fuelPriceList := datascrape.ScrapeDateAndFuelPrices(doc)

	// Take each row from the database
	rows, err := database.Query("SELECT * from prices ORDER BY timestamp ASC")
	if err != nil {
		fmt.Println("Query has failed: ", err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var date int64
		var brent float64
		var diesel float64
		var exchange float64
		err = rows.Scan(&date, &brent, &diesel, &exchange)
		if err != nil {
			fmt.Println("Scan has failed: ", err)
		}
		if i+1 < len(fuelPriceList) {
			if (fuelPriceList[i].Date <= date) && (fuelPriceList[i+1].Date > date) {
				_, err := database.Exec("UPDATE prices SET fuelprice = $1 WHERE timestamp = $2", fuelPriceList[i].Diesel, date)
				if err != nil {
					fmt.Println("Update has failed: ", err)
				}
			} else {
				i++
				_, err := database.Exec("UPDATE prices SET fuelprice = $1 WHERE timestamp = $2", fuelPriceList[i].Diesel, date)
				if err != nil {
					fmt.Println("Update has failed: ", err)
				}
			}
		} else if i+1 == len(fuelPriceList) {
			_, err := database.Exec("UPDATE prices SET fuelprice = $1 WHERE timestamp = $2", fuelPriceList[i].Diesel, date)
			if err != nil {
				fmt.Println("Insert has failed: ", err)
			}
		}
	}
}

func UpdateTableFuelPrice() {

}

func FillTableExchangeRate() {
	// Connect to database
	database := database.Connect()
	defer database.Close()

	// Get usd exchange rate from bloomberght.com
	usdExchangeRate := datascrape.GetUSDExchangeRate()

	var timestamp int64
	var price float64
	var fuelPrice float64
	var usdExchangeRateColumn float64

	// Take each row from the database
	rows, err := database.Query("SELECT * from prices ORDER BY timestamp ASC")
	if err != nil {
		fmt.Println("Query has failed: ", err)
	}
	defer rows.Close()

	// As I know from API, the data's timestamp is same as brentoilprices.
	// So I can use the timestamp directly.
	for rows.Next() {
		scanError := rows.Scan(&timestamp, &price, &fuelPrice, &usdExchangeRateColumn)
		if scanError != nil {
			fmt.Println("Scan has failed: ", scanError)
		}
		for v := range usdExchangeRate {
			if timestamp == usdExchangeRate[v].Timestamp {
				_, updateError := database.Exec("UPDATE prices SET exchangerate = $1 WHERE timestamp = $2", usdExchangeRate[v].Price, timestamp)
				if updateError != nil {
					fmt.Println("Update has failed: ", updateError)
				}
				break
			}
		}
	}
	// Some of the exchangerate datas are missing.
	// So I will fill the missing ones with the previous day's price.
	// Take each row from the database
	exchangeRow, err := database.Query("SELECT timestamp FROM prices WHERE exchangerate=0 ORDER BY timestamp")
	if err != nil {
		fmt.Println("Query has failed: ", err)
	}
	defer exchangeRow.Close()
	v := 0
	for exchangeRow.Next() {
		var timestamp int64

		scanError := exchangeRow.Scan(&timestamp)
		if scanError != nil {
			fmt.Println("Scan has failed: ", scanError)
		}
		if usdExchangeRate[v].Timestamp >= timestamp {
			_, err := database.Exec("UPDATE prices SET exchangerate = $1 WHERE timestamp = $2", usdExchangeRate[v].Price, timestamp)
			if err != nil {
				fmt.Println("Error inserting brentoilprice:", err)
				return
			}
		} else {
			for usdExchangeRate[v].Timestamp < timestamp {
				v++
			}
			_, err := database.Exec("UPDATE prices SET exchangerate = $1 WHERE timestamp = $2", usdExchangeRate[v].Price, timestamp)
			if err != nil {
				fmt.Println("Error inserting brentoilprice:", err)
				return
			}
		}
	}

}

func UpdateTableExchangeRate() {

}
func writeHeader(file *os.File) {
	header := []string{"BrentOilPrice", "FuelPrice", "USD/TRY"}
	for _, v := range header {
		if v == "USD/TRY" {
			file.WriteString(v + "\n")

		} else {
			file.WriteString(v + ",")
		}
	}
}

func CreateAndWritetoCSV() {
	// Connect to database
	database := database.Connect()

	rows, _ := database.Query("SELECT * FROM prices ORDER BY timestamp ASC")
	defer rows.Close()

	file, _ := os.Create("cleanData.csv")
	defer file.Close()

	// Write the header
	writeHeader(file)

	for rows.Next() {
		var timestamp int64
		var price float64
		var fuelPrice float64
		var usdExchangeRateColumn float64
		scanError := rows.Scan(&timestamp, &price, &fuelPrice, &usdExchangeRateColumn)
		if scanError != nil {
			panic(scanError)
		}
		file.WriteString(strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(fuelPrice, 'f', -1, 64) + "," + strconv.FormatFloat(usdExchangeRateColumn, 'f', -1, 64) + "\n")
	}
}

func UpdateCSV() {
	//Connect to database
	database := database.Connect()

	rows, _ := database.Query("SELECT * FROM prices ORDER BY timestamp ASC")
	defer rows.Close()

	file, _ := os.OpenFile("cleanData.csv", os.O_RDWR, 0666)
	defer file.Close()

	// Let's delete the old data in the file
	file.Truncate(0)

	// Write the header
	writeHeader(file)

	for rows.Next() {
		var timestamp int64
		var brentPrice float64
		var fuelPrice float64
		var usdExchangeRateColumn float64
		scanError := rows.Scan(&timestamp, &brentPrice, &fuelPrice, &usdExchangeRateColumn)
		if scanError != nil {
			panic(scanError)
		}
		file.WriteString(strconv.FormatFloat(brentPrice, 'f', -1, 64) + "," + strconv.FormatFloat(fuelPrice, 'f', -1, 64) + "," + strconv.FormatFloat(usdExchangeRateColumn, 'f', -1, 64) + "\n")
	}
}
