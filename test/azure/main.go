package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"encoding/json"

	"golang.org/x/oauth2/clientcredentials"
)

type User struct {
	Ctxt   string       `json:"@odata.context"`
	Value  []UserValue  `json:"value"`
}

type UserValue struct {
	DisplayName			string    `json:"displayName"`
	GivenName			string    `json:"givenName"`
	Surname				string    `json:"surname"`
	Mail				string    `json:"mail"`
	UserPrincipalName	string    `json:"userPrincipalName"`
	Id					string    `json:"id"`
}

func UsersGraphURL(suffix string, query map[string]string) (string, error) {
	u, err := url.Parse("https://graph.microsoft.com/v1.0/users" + suffix)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for k,v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func main() {
	//tenant := "50e3c41c-79b1-45e2-bc89-e119ffc7c692"
	tenant := "hive13members.onmicrosoft.com"
	token_url := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
		tenant)

	// Get OAuth token:
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

	// Build request URL for given badge number:
	badge_id := "178563"
	rq_user_url, err := UsersGraphURL("", map[string]string {
		"$filter": "extension_243b36ba13e74e8e86cdbb23779f1e1a_hiveBadgeId eq " + badge_id,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using request URL: %s", rq_user_url)

	// Query user from this:
	resp, err := client.Get(rq_user_url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Response: %s", resp.Status)
	}
	
	log.Printf("resp=%+v", resp)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Body: %s", b)

	var user User
	err = json.Unmarshal(b, &user)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("User: %+v", user)

	// Try to query group membership:
	membership_url, err := UsersGraphURL(
		"/" + user.Value[0].Id + "/transitiveMemberOf/microsoft.graph.group/",
		map[string]string {
			"$count": "true",
			"$select": "id,displayName",
			// TODO: Figure out why the below gives 400 Bad Request
			// "$search": "\"displayName:Active Members\"",
		})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Using request URL: %s", membership_url)
	resp, err = client.Get(membership_url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Response: %s", resp.Status)
	}

	log.Printf("resp=%+v", resp)
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Body: %s", b)
}
