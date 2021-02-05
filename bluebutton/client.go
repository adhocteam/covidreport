/*
Copyright 2021 Ad Hoc LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package bluebutton

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BBClientID     string
	BBClientSecret string
	BBURL          string
	CallbackURL    string
}

func (c *Client) String() string {
	var truncatedSecret string
	if len(c.BBClientSecret) > 5 {
		truncatedSecret = c.BBClientSecret[:5] + "..."
	} else {
		truncatedSecret = "<empty>"
	}

	return fmt.Sprintf("Blue Button Client {%s %s %s %s}",
		c.BBClientID,
		truncatedSecret,
		c.BBURL,
		c.CallbackURL)
}

func makeRandomState() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("Unable to get random numbers")
	}
	return fmt.Sprintf("%x", b)
}

func (c *Client) AuthURL() string {
	// should url-encode the params
	return fmt.Sprintf("%s/v1/o/authorize/?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		c.BBURL, c.BBClientID, c.CallbackURL, makeRandomState())
}

// FullToken represents the "full token" returned by blue button
// https://bluebutton.cms.gov/developers/#web-application-flow
type FullToken struct {
	AccessToken  string  `json:"access_token"`
	Expires      float32 `json:"expires_in"`
	TokenType    string  `json:"token_type"`
	Scope        string  `json:"scope"`
	RefreshToken string  `json:"refresh_token"`
}

func (c *Client) requestFullToken(url string, data url.Values) (*FullToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("getting %s", url)

	client := &http.Client{}
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// XXX: necessary?
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	req.SetBasicAuth(c.BBClientID, c.BBClientSecret)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Request took: %s", time.Since(start))

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200, got %v", resp.Status)
	}

	var fullToken FullToken
	err = json.Unmarshal(respBody, &fullToken)
	if err != nil {
		return nil, err
	}

	return &fullToken, nil
}

// get accepts a url and authorization token for a Blue Button API call. It
// will attempt to unmarshal the response into `obj`
//
// curl --header "Authorization: Bearer YOUR TOKEN HERE"
// https://sandbox.bluebutton.cms.gov/v1/fhir/Patient/-20140000008325
//
// curl --header "Authorization: Bearer AUTHORIZATION TOKEN"
// "https://sandbox.bluebutton.cms.gov/v1/connect/userinfo"
//
// TODO: To activate compression add the following to the header:
//
// Accept-Encoding: gzip
//
// does go add this by default? I forget. Can we just add it?
func get(url, tok string, obj interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("getting %s", url)

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tok))

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("Request took: %s", time.Since(start))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200, got %v", resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(respBody, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetFullToken(callbackToken string) (*FullToken, error) {
	// https://golang.cafe/blog/how-to-make-http-url-form-encoded-request-golang.html
	params := url.Values{}
	params.Set("code", callbackToken)
	params.Set("grant_type", "authorization_code")
	params.Set("redirect_uri", c.CallbackURL)
	fullTokenURL := fmt.Sprintf("%s/v1/o/token/", c.BBURL)
	return c.requestFullToken(fullTokenURL, params)
}

// https://bluebutton.cms.gov/developers/#core-resources
// {
//   "sub": "fflinstone",
//   "prefered_username": "fflinstone",
//   "given_name": "Fred",
//   "family_name:, "Flinstone,
//   "name": "Fred Flinstone",
//   "email": "pebbles-daddy@example.com",
//   "created": "2017-11-28",
//   "patient": "123456789",
// }
type UserInfo struct {
	Sub              string `json:"sub"`
	PreferedUsername string `json:"prefered_username"`
	GivenName        string `json:"given_name"`
	FamilyName       string `json:"family_name"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	Created          string `json:"created"`
	FhirID           string `json:"patient"`
}

func (c *Client) GetUserInfo(accessToken string) (*UserInfo, error) {
	var user UserInfo
	err := get(fmt.Sprintf("%s/v1/connect/userinfo", c.BBURL), accessToken, &user)
	return &user, err
}

// this type is used to parse dates of the format year-month-day
type YearMonthDay struct {
	Time time.Time
}

func (j YearMonthDay) Format(pat string) string {
	return j.Time.Format(pat)
}

func (j *YearMonthDay) UnmarshalJSON(b []byte) error {
	tm, err := time.Parse("2006-01-02", strings.Trim(string(b), "\""))
	j.Time = tm
	return err
}

// $ curl --header 'Authorization: Bearer <tok>' \
//   'https://sandbox.bluebutton.cms.gov/v1/fhir/Patient/-19990000000001'
// "resourceType": "Patient",
// "id": "-19990000000001",
// "meta": {
//   "lastUpdated": "2020-11-09T22:49:27.580+00:00"
// },
// "extension": [...omitted... ],
// "identifier": [...omitted... ],
// "name": [
//   {
//     "use": "usual",
//     "family": "Doe",
//     "given": [
//       "Jane",
//       "X"
//     ]
//   }
// ],
// "gender": "female",
// "birthDate": "1999-06-01",
// "address": [
//   {
//     "district": "999",
//     "state": "30",
//     "postalCode": "99999"
//   }
// ]
type PatientName struct {
	Use    string   `json:"use"`
	Family string   `json:"family"`
	Given  []string `json:"given"`
}

type Patient struct {
	Meta struct {
		LastUpdated string `json:"lastUpdated"`
	} `json:"meta"`
	Name      []PatientName
	Gender    string       `json:"gender"`
	BirthDate YearMonthDay `json:"birthDate"`
	Address   []struct {
		District   string `json:"district"`
		State      string `json:"state"`
		PostalCode string `json:"postalCode"`
	} `json:"address"`
}

func (c *Client) GetPatient(fhirID, accessToken string) (*Patient, error) {
	var pat Patient
	err := get(fmt.Sprintf("%s/v1/fhir/Patient/%s", c.BBURL, fhirID), accessToken, &pat)
	return &pat, err
}
