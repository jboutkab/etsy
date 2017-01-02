package main

import (
	"github.com/dghubble/oauth1"

	"fmt"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

//1d9d5612

var config oauth1.Config

func main() {
	var log = logging.MustGetLogger("etsy")

	configviper := viper.New()
	configviper.SetDefault("etsy.ConsumerKey", "gggggggg")
	configviper.SetDefault("etsy.ConsumerSecret", "ggggggg")
	configviper.SetDefault("etsy.CallbackURL", "oob")
	//configviper.SetDefault("etsy.Endpoint",etsyEndpoint)
	configviper.SetDefault("etsy.RequestTokenURL", "https://openapi.etsy.com/v2/oauth/request_token")
	configviper.SetDefault("etsy.AccessTokenURL", "https://openapi.etsy.com/v2/oauth/access_token")

	//Parse configuration file
	configviper.SetConfigName("config") // name of config file (without extension)
	configviper.AddConfigPath(".")      // path to look for the config file in

	err := configviper.ReadInConfig()
	if err != nil {
		log.Fatal("Config file not found in current directory...")
	} else {

		log.Info("Config found...")
	}

	consumerKey := configviper.GetString("etsy.ConsumerKey")
	consumerSecret := configviper.GetString("etsy.ConsumerSecret")
	callbackURL := configviper.GetString("etsy.CallbackURL")
	requestTokenURL := configviper.GetString("etsy.RequestTokenURL")
	accessTokenURL := configviper.GetString("etsy.AccessTokenURL")

	ßetsyEndpoint := oauth1.Endpoint{RequestTokenURL: requestTokenURL, AccessTokenURL: accessTokenURL}

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
	requestToken, requestSecret, err = config.RequestToken()
	if err != nil {
		return "", "", err
	}
	authorizationURLSuffix, err := config.AuthorizationURL(requestToken)
	if err != nil {
		return "", "", err
	}
	authorizationURL := fmt.Sprintf("https://www.etsy.com/oauth/signin%s", authorizationURLSuffix)
	fmt.Printf("Open this URL in your browser:\n%s\n", authorizationURL)
	return requestToken, requestSecret, err
}
