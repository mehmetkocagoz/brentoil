package datascrape

import (
	"fmt"
	"mehmetkocagz/database"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FuelPrice struct {
	Date   int64
	Diesel float64
}

// Get fuel prices from tppd.com.tr
// This function will just return the fuel prices as a document.
func GetFuelPrices() *goquery.Document {
	url := "https://www.tppd.com.tr/en/former-oil-prices?id=35&county=429&StartDate=22.11.2018&EndDate=22.08.2023"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Get request has failed: ", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("html.Parse has failed: ", err)
	}
	return doc
}

func ScrapeDateAndFuelPrices(doc goquery.Document) []FuelPrice {
	var fuelPrices []FuelPrice
	var date int64
	var diesel float64
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		s.Find("td").Each(func(i int, s *goquery.Selection) {
			// As I know how data table structered I can get the data I want.
			// I'm going to get the date and DIESEL prices only.
			// This function can be improved.
			if i == 0 {
				// Date will come to us as string. 13 May 2019
				// I'm going to convert it to int64.
				date = convertTimestamp(s.Text())

			} else if i == 4 {
				// Assume there is no error.
				// We can handle it later.
				diesel, _ = strconv.ParseFloat(s.Text(), 64)
			}
		})
		fuelPrices = append(fuelPrices, FuelPrice{date, diesel})
	})
	return fuelPrices
}

func switchMonthToNumber(month string) string {
	switch month {
	case "January":
		return "01"
	case "february":
		return "02"
	case "March":
		return "03"
	case "April":
		return "04"
	case "May":
		return "05"
	case "June":
		return "06"
	case "July":
		return "07"
	case "August":
		return "08"
	case "September":
		return "09"
	case "October":
		return "10"
	case "November":
		return "11"
	case "December":
		return "12"
	}
	return "0"
}

func convertTimestamp(date string) int64 {
	fmt.Println("converting..", date)
	// I know that our date will come like int string int format.
	// So first I'm going to convert it to int-int-int format.
	parsedDate := strings.Split(date, " ")
	month := switchMonthToNumber(parsedDate[1])
	date = parsedDate[2] + "-" + month + "-" + parsedDate[0]
	layout := "2006-01-02"
	t, err := time.Parse(layout, date)

	if err != nil {
		fmt.Println("time.Parse has failed: ", err)
	}
	return (t.Unix() * 1000)
}

// TODO: Insert fuel prices to database.
func InsertFuelPrices(dataList []FuelPrice) {
	//When examining the data from the website, I noticed that the data isn't updated
	//on a daily basis; instead, it is updated whenever new data arrives. Since I want to
	//utilize the daily price changes
	//I'm going to insert the data into the database on a daily basis.
	//I will apply pricing policies for days that are not listed based on the previous data.
	//I will also apply pricing policies for days that are listed but have no data.
	//In the beginning, it will be one time job so I'm not going to afraid of performance issues.
	db := database.Connect()
	defer db.Close()

	// Take each row from the database
	rows, err := db.Query("SELECT * from pricedata")
	if err != nil {
		fmt.Println("Query has failed: ", err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var date int64
		var brent float64
		var diesel float64
		err = rows.Scan(&date, &brent, &diesel)
		if err != nil {
			fmt.Println("Scan has failed: ", err)
		}

		for (dataList[i].Date < date) && (dataList[i].Date > date) {
			_, err := db.Exec("UPDATE pricedata SET fuelprice = $1 WHERE timestamp = $2", dataList[i].Diesel, date)
			rows.Next()
			if err != nil {
				fmt.Println("Insert has failed: ", err)
			}
		}
		i++
	}
}
