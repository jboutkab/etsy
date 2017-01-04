package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	_ "strconv"
	"strings"
	"time"
)

type Image struct {
	Url_75x75     string
	Url_170x135   string
	Url_570xN     string
	Url_fullxfull string
	Full_height   int
	Full_width    int
}

type Transactions struct {
	Count   int
	Results []Transaction
}

type Transaction struct {
	Transaction_id int
	Receipt_id     int
	Creation_tsz   int64
	Paid_tsz       int64
	Shipped_tsz    int64
	Price          string
	Currency_code  string
	Quantity       int
	Tags           []string
	Listing_id     int
}

type Credentials struct {
	appname        string
	ConsumerKey    string
	ConsumerSecret string
	token          string
	tokenSecret    string
}

type Listing struct {
	Url         string
	Listing_id  int
	State       string
	User_id     int
	Title       string
	Description string
	Images      []Image
}

type Listings struct {
	Count   int
	Results []Listing
}

type Etsy struct {
	apiKey string
}

var transaction_url = "https://openapi.etsy.com/v2/shops/__SELF__/transactions?fields=transaction_id,receipt_id,creation_tsz,paid_tsz,shipped_tsz,price,currency_code,quantity,tags,listing_id&limit=200"
var api_url = "https://openapi.etsy.com/v2/"
var etsydb = "/Users/jimmies/etsy-github/etsy/etsy.db"
var name string
var ConsumerKey string
var ConsumerSecret string
var token string
var tokenSecret string

func New(apiKey string) *Etsy {
	return &Etsy{
		apiKey: apiKey,
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func CheckErrPrint(err error) {
	if err != nil {
		log.Println(err)

	}
}

func (e *Etsy) authenticate(url string) string {
	return url + "&api_key=" + e.apiKey
}

func (e *Etsy) GetStoreListings(limit int) (Listings, error) {

	//url := api_url + fmt.Sprintf("shops/%s/listings/active?fields=title&limit=%d", storeID, limit)
	url := api_url + fmt.Sprintf("listings/active?fields=title,listing_id,state&limit=%d", limit)
	url = e.authenticate(url)
	fmt.Println(url)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return Listings{}, err
	}

	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
		return Listings{}, err
	}

	// marshal into structs
	var listings Listings
	json.Unmarshal([]byte(result), &listings)

	return listings, nil
}

func GetStoreTransactions(credentials []Credentials) (Transactions, error) {

	config := oauth1.NewConfig(credentials[0].ConsumerKey, credentials[0].ConsumerSecret)
	token := oauth1.NewToken(credentials[0].token, credentials[0].tokenSecret)
	httpclient := config.Client(oauth1.NoContext, token)

	res, err := httpclient.Get(transaction_url)
	if err != nil {

		log.Fatal(err)
	}

	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatal(err)

	}
	//fmt.Printf("Raw Response Body:\n%v\n", string(result))
	var transactions Transactions
	json.Unmarshal([]byte(result), &transactions)

	return transactions, nil
}

func OauthRetrieve(appname string, dbname string) ([]Credentials, error) {

	db, err := sql.Open("sqlite3", dbname)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query("select appname,ConsumerKey,ConsumerSecret,token,tokenSecret from apikey where appname=?", appname)

	checkErr(err)
	defer rows.Close()

	credentials := []Credentials{}
	for rows.Next() {
		var cred Credentials

		err := rows.Scan(&cred.appname, &cred.ConsumerKey, &cred.ConsumerSecret, &cred.token, &cred.tokenSecret)
		checkErr(err)
		fmt.Printf("%s,%s,%s,%s,%s", cred.appname, cred.ConsumerKey, cred.ConsumerSecret, cred.token, cred.tokenSecret)
		credentials = append(credentials, cred)

	}
	return credentials, nil

}

func main() {
	store := flag.String("store", "mybebecadum", "store name to get oauth")
	flag.Parse()

	configviper := viper.New()
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.appname", *store), "jamaltest")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.dbname", *store), "etsy.db")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.transactiontable", *store), "transactions")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.stockstable", *store), "transactions")

	configviper.SetConfigName("config") // name of config file (without extension)
	configviper.AddConfigPath(".")      // path to look for the config file in

	err := configviper.ReadInConfig()
	if err != nil {
		log.Fatal("Config file not found in current directory...")
	} else {

		log.Print("Config found...")
	}

	appname := configviper.GetString(fmt.Sprintf("%s.etsy-db.appname", *store))
	dbname := configviper.GetString(fmt.Sprintf("%s.etsy-db.dbname", *store))
	transactionstable := configviper.GetString(fmt.Sprintf("%s.etsy-db.transactiontable", *store))
	stockstable := configviper.GetString(fmt.Sprintf("%s.etsy-db.stockstable", *store))

	credentials, err := OauthRetrieve(appname, dbname)
	if err != nil {
		log.Fatal("error retrieving token", err)
	}

	t, err := GetStoreTransactions(credentials)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", dbname)
	checkErr(err)
	defer db.Close()
	for o, _ := range t.Results {

		//check
		c := time.Unix(t.Results[o].Creation_tsz, 0)
		now := time.Now()
		diff := now.Sub(c)
		tagsString := strings.Join(t.Results[o].Tags, ",")
		//log.Println(tagsString)
		//fmt.Println(t.Results[o].Listing_id,t.Results[o].Transaction_id,t.Results[o].Tags)
		if diff.Hours() < 48 {
			//insert transactions in DB
			log.Printf("Prepare")
			log.Println(transactionstable)

			insertStmttx := fmt.Sprintf("insert into %s  (transaction_id,listing_id,quantity,price,currency_code,creation_tsz,paid_tsz,shipped_tsz,tags) values(?,?,?,?,?,?,?,?,?)", transactionstable)
			stmttx, err := db.Prepare(insertStmttx)
			log.Printf("insert tx")

			_, err = stmttx.Exec(t.Results[o].Transaction_id, t.Results[o].Listing_id, t.Results[o].Quantity, t.Results[o].Price, t.Results[o].Currency_code, t.Results[o].Creation_tsz, t.Results[o].Paid_tsz, t.Results[o].Shipped_tsz, tagsString)
			fmt.Println(err)
			if err == nil {
				log.Println("udpate stocks table")
				updatStmtstk := fmt.Sprintf("Update %s set soldtodate=soldtodate + ? where skuname=?", stockstable)
				if strings.Contains(tagsString, "Tote") {
					stmttk, err := db.Prepare(updatStmtstk)
					checkErr(err)
					_, err = stmttk.Exec(t.Results[o].Quantity, "tote")

					fmt.Println(err, t.Results[o].Transaction_id)

				} else if strings.Contains(tagsString, "Large") {
					stmttk, err := db.Prepare(updatStmtstk)
					checkErr(err)
					_, err = stmttk.Exec(t.Results[o].Quantity, "largecosmetic")

					fmt.Println(err, t.Results[o].Transaction_id)

				} else {

					stmttk, err := db.Prepare(updatStmtstk)
					checkErr(err)
					_, err = stmttk.Exec(t.Results[o].Quantity, "smallcosmetic")

					fmt.Println(err, t.Results[o].Transaction_id)

				}

			}
			/*if err !:=nil {
				log.Println("error insertStmttx",err)
				break
			} */
			/*insertStmtstk := fmt.Sprintf("insert or ignore into %s (productname,details,stock,sold,available) VALUES (?,?,?,?,?)", stockstable)
			log.Println(insertStmtstk)
			stmttk, err := db.Prepare(insertStmtstk)
			log.Printf("je suis ici")

			_, err = stmttk.Exec()
			log.Printf("je suis ici1")
			log.Println(err)

			checkErr(err)

			updatStmtstk := fmt.Sprintf("Update %s set sold=sold + ? where productname=tote", stockstable)
			stmttk, err = db.Prepare(updatStmtstk)
			checkErr(err)
			_, err = stmttk.Exec(t.Results[o].Quantity)
			checkErr(err)*/
			println(stockstable)

		}

		/*for j, _ := range t.Results[o].Tags {

			fmt.Println(t.Results[o].Transaction_id, t.Results[o].Receipt_id, t.Results[o].Quantity, t.Results[o].Price, time.Unix(t.Results[o].Creation_tsz, 0))

			fmt.Println(t.Results[o].Tags[j])

		}*/
	}
}
