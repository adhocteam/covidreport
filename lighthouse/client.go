package lighthouse

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
	ClientID     string
	ClientSecret string
	URL          string
	FhirURL      string
	CallbackURL  string
}

func (c *Client) String() string {
	var truncatedSecret string
	if len(c.ClientSecret) > 5 {
		truncatedSecret = c.ClientSecret[:5] + "..."
	} else {
		truncatedSecret = "<empty>"
	}

	return fmt.Sprintf("VA Lighthouse Client {%s %s %s %s %s}",
		c.ClientID,
		truncatedSecret,
		c.URL,
		c.CallbackURL,
		c.FhirURL)
}

func makeRandomState() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("Unable to get random numbers")
	}
	return fmt.Sprintf("%x", b)
}

func (c Client) AuthURL(scope string) string {
	return fmt.Sprintf("%s/oauth2/authorization?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=%s",
		c.URL, c.ClientID, c.CallbackURL, makeRandomState(), scope)
}

// {
//   "access_token": "SlAV32hkKG",
//   "expires_in": 3600,
//   "refresh_token": "8xLOxBtZp8",
//   "scope": "openid profile email offline_access",
//   "patient": "1558538470",
//   "state": "af0ifjsldkj",
//   "token_type": "Bearer",
// }
type FullToken struct {
	AccessToken  string  `json:"access_token"`
	Expires      float32 `json:"expires_in"`
	TokenType    string  `json:"token_type"`
	Scope        string  `json:"scope"`
	RefreshToken string  `json:"refresh_token"`
	State        string  `json:"state"`
	PatientID    string  `json:"patient"`
}

func (c Client) GetFullToken(callbackToken, state string) (*FullToken, error) {
	params := url.Values{}
	params.Set("code", callbackToken)
	params.Set("grant_type", "authorization_code")
	params.Set("redirect_uri", c.CallbackURL)
	params.Set("state", state)
	fullTokenURL := fmt.Sprintf("%s/oauth2/token/", c.URL)
	return c.requestFullToken(fullTokenURL, params)
}

func (c *Client) requestFullToken(url string, data url.Values) (*FullToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("sending %v to %s", data, url)

	client := &http.Client{}
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// XXX: necessary?
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	req.SetBasicAuth(c.ClientID, c.ClientSecret)

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

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200, got %s", resp.Status)
	}

	err = json.Unmarshal(respBody, obj)
	if err != nil {
		return err
	}
	return nil
}

type Code struct {
	System  string      `json:"system,omitempty"`
	Code    string      `json:"code,omitempty"`
	Display string      `json:"display,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

type ImmunizationResource struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
	Status       string `json:"status"`
	VaccineCode  struct {
		Coding []Code `json:"coding"`
	} `json:"vaccineCode"`
	Text    string `json:"text"`
	Patient struct {
		Reference string `json:"reference"`
		Display   string `json:"display"`
	}
	Occurence string `json:"occurrenceString"`
	Reaction  []struct {
		Detail struct {
			Display string `json:"display"`
		} `json:"detail"`
	} `json:"reaction"`
}

type ImmunizationResponse struct {
	Links []struct {
		Relation string `json:"relation"`
		Url      string `json:"url"`
	} `json:"link"`
	Entries []struct {
		FullURL  string               `json:"fullUrl"`
		Resource ImmunizationResource `json:"resource"`
	} `json:"entry"`
}

type Vaccination struct {
	Date     time.Time
	Code     string
	Display  string
	Location string
	Lot      string
}

// todo return some sort of more useful vaccination struct. returning the immunizationresponse for now just to test things out
func (c Client) GetVaccinations(tok, patientID string) ([]Vaccination, error) {
	if patientID == "" {
		return nil, fmt.Errorf("invalid patient id")
	}
	var res ImmunizationResponse
	// TODO add paging
	err := get(fmt.Sprintf("%s/Immunization?patient=%s", c.FhirURL, patientID), tok, &res)
	if err != nil {
		return nil, err
	}

	// TODO search through the vaccination response for a covid vaccination
	var vaxes []Vaccination
	return vaxes, nil
}

type PatientResponse struct {
	Links []struct {
		Relation string `json:"relation"`
		Url      string `json:"url"`
	} `json:"link"`
	Names []struct {
		Use    string   `json:"use"`
		Text   string   `json:"text"`
		Family string   `json:"family"`
		Given  []string `json:"given"`
	} `json:"name"`
	BirthDate YearMonthDay `json:"birthDate"`
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

type Patient struct {
	Name      string
	BirthDate YearMonthDay
}

func (c Client) GetPatient(tok, patientID string) (*Patient, error) {
	if patientID == "" {
		return nil, fmt.Errorf("invalid patient id")
	}
	var res PatientResponse
	err := get(fmt.Sprintf("%s/Patient/%s", c.FhirURL, patientID), tok, &res)
	if err != nil {
		return nil, err
	}
	pat := &Patient{
		Name:      res.Names[0].Text,
		BirthDate: res.BirthDate,
	}
	return pat, nil
}
