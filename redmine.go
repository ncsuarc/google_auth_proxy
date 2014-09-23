package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Redmine struct {
	BaseURL string
	ApiKey  string
}

type RedmineUser struct {
	Id        int       `json:"id"`
	FirstName string    `json:"firstname"`
	LastName  string    `json:"lastname"`
	CreatedOn time.Time `json:"created_on"`
	LastLogin time.Time `json:"last_login_on"`
	Email     string    `json:"mail"`
}

type RedmineUsersEndpoint struct {
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	TotalCount int           `json:"total_count"`
	Users      []RedmineUser `json:"users"`
}

func (r *Redmine) getEndpoint(endpoint string) ([]byte, error) {
	url := r.BaseURL + endpoint

	log.Println("GET " + url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte(""), err
	}

	req.Header.Add("X-Redmine-API-Key", r.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []byte(""), err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), err
	}

	return b, nil
}

func (r *Redmine) Users(params url.Values) (*RedmineUsersEndpoint, error) {
	var e RedmineUsersEndpoint

	b, err := r.getEndpoint("/users.json?" + params.Encode())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &e)

	return &e, nil
}

func (r *Redmine) AllUsers(name string, group int) (*[]RedmineUser, error) {
	var all_users []RedmineUser

	offset := 0

	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	if group >= 0 {
		params.Set("group_id", strconv.Itoa(group))
	}

	for {
		params.Set("offset", strconv.Itoa(offset))

		users, err := r.Users(params)
		if err != nil {
			return nil, err
		}

		all_users = append(all_users, users.Users...)

		if len(users.Users) == 0 {
			break
		}

		if users.Offset+users.Limit >= users.TotalCount {
			break
		}

		offset += users.Limit
	}

	return &all_users, nil
}

func (r *Redmine) UserInGroup(email string, group int) (bool, error) {
	users, err := r.AllUsers(email, group)
	if err != nil {
		return false, err
	}

	for _, user := range *users {
		if user.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func NewRedmine(base string, key string) (*Redmine, error) {
	return &Redmine{
		BaseURL: base,
		ApiKey:  key,
	}, nil
}

func NewRedmineValidator(base_url string, key string, group int) func(string) bool {
	redmine, err := NewRedmine(base_url, key)
	if err != nil {
		log.Fatalf("Failed to create Redmine struct, %s", err.Error())
	}

	validator := func(email string) bool {
		valid, err := redmine.UserInGroup(email, group)
		if err != nil {
			log.Printf("Failed to lookup user, %s", err.Error())
			return false
		}

		return valid
	}

	return validator
}
