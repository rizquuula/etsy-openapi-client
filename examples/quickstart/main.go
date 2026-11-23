// Command quickstart calls the authenticated Etsy getMe endpoint.
//
// Required environment variables:
//
//	ETSY_KEYSTRING
//	ETSY_SHARED_SECRET
//	ETSY_ACCESS_TOKEN
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func main() {
	client, err := etsy.New(etsy.Config{
		Keystring:    os.Getenv("ETSY_KEYSTRING"),
		SharedSecret: os.Getenv("ETSY_SHARED_SECRET"),
		AccessToken:  os.Getenv("ETSY_ACCESS_TOKEN"),
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GetMeWithResponse(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body); err != nil {
		log.Fatal(err)
	}
	if resp.JSON200 == nil || resp.JSON200.UserId == nil {
		log.Fatal("getMe returned an empty user")
	}
	fmt.Printf("authenticated as user %d\n", *resp.JSON200.UserId)
}
