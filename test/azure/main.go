package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	//tenant := "50e3c41c-79b1-45e2-bc89-e119ffc7c692"
	tenant := "hive13members.onmicrosoft.com"
	token_url := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
		tenant)

	user_id := "12345"
	rq_url := "https://graph.microsoft.com/v1.0/users?$filter=extension_243b36ba13e74e8e86cdbb23779f1e1a_hiveBadgeId%20eq%20" + user_id

	ctx := context.Background()
	conf := clientcredentials.Config{
		ClientSecret: client_secret,
		ClientID: client_id,
		Scopes: []string{
			"https://graph.microsoft.com/.default",
		},
		TokenURL: token_url,
	}

	log.Printf("Client ID: %s", client_id)
	log.Printf("Client secret: %s", client_secret)
	log.Printf("Tenant: %s", tenant)
	log.Printf("Using token URL: %s", token_url)

	client := conf.Client(ctx)

	log.Printf("Using user ID: %s", user_id)
	log.Printf("Using request URL: %s", rq_url)
	
	resp, err := client.Get(rq_url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Response: %s", resp.Status)
	} else {
		log.Printf("resp=%+v", resp)
	}
}
