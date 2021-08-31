package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	active_members_guid = "8194af1c-7c5e-486d-a175-c5b3d4e08792"
)

type User struct {
	Ctxt  string      `json:"@odata.context"`
	Value []UserValue `json:"value"`
}

type Groups struct {
	Ctxt  string   `json:"@odata.context"`
	Value []string `json:"value"`
}

type UserValue struct {
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	Id                string `json:"id"`
}

type GroupQuery struct {
	GroupIds []string `json:"groupIds"`
}

func UsersGraphURL(suffix string, query map[string]string) (string, error) {
	u, err := url.Parse("https://graph.microsoft.com/v1.0/users" + suffix)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for k, v := range query {
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
		ClientID:     client_id,
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
	rq_user_url, err := UsersGraphURL("", map[string]string{
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

	groupQuery := GroupQuery{
		GroupIds: []string{active_members_guid},
	}

	var jsonGroupQuery []byte
	jsonGroupQuery, err = json.Marshal(groupQuery)
	if err != nil {
		log.Fatal(err)
	}

	membership_url, err := UsersGraphURL("/"+user.Value[0].Id+"/checkMemberGroups", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Using request URL: %s", membership_url)
	resp, err = client.Post(membership_url, "application/json", bytes.NewBuffer(jsonGroupQuery))
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

	var user_groups Groups
	err = json.Unmarshal(b, &user_groups)
	if err != nil {
		log.Fatal(err)
	}

	// Returned array should have group UUID if user is a member. Empty mean not a member
	if len(user_groups.Value) <= 0 {
		log.Fatal("Not in [Active Members] group")
	}

	log.Printf("SUCCESS: User Groups: %+v", user_groups)
}
