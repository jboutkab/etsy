package main

import (
	"fmt"
	"github.com/dghubble/oauth1"
	"github.com/spf13/viper"
	//"time"
	"database/sql"
	"flag"
	"log"
	_ "github.com/mattn/go-sqlite3"
)

var config oauth1.Config

type accessToken struct {
}

func main() {

	store := flag.String("store", "mybebecadum", "store name to get oauth")
	flag.Parse()

	configviper := viper.New()
	configviper.SetDefault(fmt.Sprintf("%s.etsy-oauth.ConsumerKey", *store), "gggggggg")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-oauth.ConsumerSecret", *store), "ggggggg")
	configviper.SetDefault(fmt.Sprintf("%setsy-oauth.CallbackURL", *store), "oob")
	//configviper.SetDefault("etsy.Endpoint",etsyEndpoint)
	configviper.SetDefault(fmt.Sprintf("%s.etsy-oauth.RequestTokenURL", *store), "https://openapi.etsy.com/v2/oauth/request_token")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-oauth.AccessTokenURL", *store), "https://openapi.etsy.com/v2/oauth/access_token")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.dbname", *store), "etsy.db")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.appname", *store), "jamatest")
	configviper.SetDefault(fmt.Sprintf("%s.etsy-db.apikeytable", *store), "apikey")

	//Parse configuration file
	configviper.SetConfigName("config") // name of config file (without extension)
	configviper.AddConfigPath(".")      // path to look for the config file in

	err := configviper.ReadInConfig()
	if err != nil {
		log.Fatal("Config file not found in current directory...")
	} else {

		log.Printf("Config found...")
	}

	consumerKey := configviper.GetString(fmt.Sprintf("%s.etsy-oauth.ConsumerKey", *store))
	consumerSecret := configviper.GetString(fmt.Sprintf("%s.etsy-oauth.ConsumerSecret", *store))
	callbackURL := configviper.GetString(fmt.Sprintf("%setsy-oauth.CallbackURL", *store))
	requestTokenURL := configviper.GetString(fmt.Sprintf("%s.etsy-oauth.RequestTokenURL", *store))
	accessTokenURL := configviper.GetString(fmt.Sprintf("%s.etsy-oauth.AccessTokenURL", *store))
	dbname := configviper.GetString(fmt.Sprintf("%s.etsy-db.dbname", *store))
	appname := configviper.GetString(fmt.Sprintf("%s.etsy-db.appname", *store))
	apikeytable := configviper.GetString(fmt.Sprintf("%s.etsy-db.apikeytable", *store))


	log.Printf(accessTokenURL,consumerKey)

	etsyEndpoint := oauth1.Endpoint{RequestTokenURL: requestTokenURL, AccessTokenURL: accessTokenURL}

	fmt.Println(etsyEndpoint.AccessTokenURL)

	config = oauth1.Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		CallbackURL:    callbackURL,
		Endpoint:       etsyEndpoint,
	}

	requestToken, requestSecret, err := login()
	if err != nil {
		log.Fatalf("Request Token Phase: %s", err.Error())
	}


	accessToken, err := receivePIN(requestToken, requestSecret)
	if err != nil {
		log.Fatalf("Access Token Phase: %s", err.Error())
	}

	fmt.Println("Consumer was granted an access token to act on behalf of a user.")
	fmt.Printf("token: %s\nsecret: %s\n", accessToken.Token, accessToken.TokenSecret)

	err = updateapikeydb(*store, appname, consumerKey, consumerSecret, accessToken, dbname, apikeytable)
	checkErr(err)

}

func updateapikeydb(store string, appname string, consumerkey string, consumersecret string, accesstoken *oauth1.Token, dbname string, table string) error {
log.Printf("Open DB")
log.Printf(dbname)
fmt.Println(store,appname,consumerkey,consumersecret,accesstoken,dbname,table)
	
	db, err := sql.Open("sqlite3", dbname)
	log.Printf("Sql.Open")
	checkErr(err)
	log.Printf("Prepare")
	stmt, err := db.Prepare("insert into apikey (store,appname,ConsumerKey,ConsumerSecret,token,tokenSecret) values(?,?,?,?,?,?)")
	
	res, err := stmt.Exec(store, appname, consumerkey, consumersecret, accesstoken.Token, accesstoken.TokenSecret)
	checkErr(err)

 	id, err := res.LastInsertId()
    checkErr(err)

    fmt.Println(id)

	defer db.Close()
	return nil

}

func receivePIN(requestToken string, requestSecret string) (*oauth1.Token, error) {
	fmt.Printf("Paste your PIN here: ")
	var verifier string
	_, err := fmt.Scanf("%s", &verifier)
	if err != nil {
		return nil, err
	}

	accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return nil, err
	}
	return oauth1.NewToken(accessToken, accessSecret), err
}

func login() (requestToken string, requestSecret string, err error) {
	log.Printf("starting the RequestToken")

	requestToken, requestSecret, err = config.RequestToken()
	if err != nil {
		return "", "", err
	}
	log.Printf("starting the authorization request")
	authorizationURLSuffix, err := config.AuthorizationURL(requestToken)
	if err != nil {
		return "", "", err
	}
	authorizationURL := fmt.Sprintf("https://www.etsy.com/oauth/signin%s", authorizationURLSuffix)
	fmt.Printf("Open this URL in your browser:\n%s\n", authorizationURL)
	return requestToken, requestSecret, err
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
