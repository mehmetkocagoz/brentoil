package main

import (
	"fmt"
	"mehmetkocagz/datascrape"
	"os/exec"
)

// This function will fill the database with the data we want.
// This function will be used only once.
// After that we will just update the database if new data comes.
func databaseFiller() {
	// Get brent oil prices from bloomberght.com
	priceList := datascrape.GetBrentOilPrices()
	// Insert brent oil prices to database
	datascrape.InsertBrentOilPrices(priceList)
	// Get fuel prices from tppd.com.tr
	doc := datascrape.GetFuelPrices()
	// Insert fuel prices to database
	datascrape.InsertFuelPrices(datascrape.ScrapeDateAndFuelPrices(*doc))
	// Update usd exchange rate
	datascrape.UpdateUSDExchangeRate()
}
func databaseUpdater() {
	// First insert new brent oil prices
	datascrape.InsertNewBrentOilPrices()
	// Then insert new fuel prices
	datascrape.InsertNewFuelPrices()
	// Update usd exchange rate
	datascrape.UpdateUSDExchangeRate()
}
func linearRegression() {
	cmd := exec.Command("python", "datafunctions/linearRegression.py")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error executing Python script:", err)
		return
	}
	fmt.Println("A", string(out))
	fmt.Println("Linear regression script executed successfully.")
}

func main() {
	databaseUpdater()
}
