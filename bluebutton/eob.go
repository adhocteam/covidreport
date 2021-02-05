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
	"fmt"
	"time"
)

type EOBResponse struct {
	Meta struct {
		LastUpdated string `json:"lastUpdated"`
	} `json:"meta"`
	Links []struct {
		Relation string `json:"relation"`
		Url      string `json:"url"`
	} `json:"link"`
	Total   int `json:"total"`
	Entries []struct {
		Resource EOBResource `json:"resource"`
	} `json:"entry"`
}

func (e EOBResponse) Next() string {
	for _, link := range e.Links {
		if link.Relation == "next" {
			return link.Url
		}
	}
	return ""
}

// Code is a structured BlueButton data element.
// It may not contain all elements.
type Code struct {
	System  string      `json:"system,omitempty"`
	Code    string      `json:"code,omitempty"`
	Display string      `json:"display,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

type Coding struct {
	Coding []Code `json:"coding"`
}

type EOBItem struct {
	ServicedDate     string `json:"servicedDate"`
	ProductOrService Coding `json:"ProductOrService"`
	Service          Coding `json:"service"`
}

type EOBResource struct {
	Status string    `json:"status"`
	Type   Coding    `json:"type"`
	Items  []EOBItem `json:"item"`
}

type EOBEntry struct {
	Resource EOBResource `json:"resource"`
}

type Vaccination struct {
	Date     time.Time
	Code     string
	Display  string
	Location string
	Lot      string
}

// Jack Williams
// https://adhoc.slack.com/archives/CVB2Y9NE5/p1610564186061500?thread_ts=1610563465.058200&cid=CVB2Y9NE5
// The easiest way i believe would to look for the HCPCS code for Covid in the
// “ExplanationOfBenefit.item.productOrService” field.
//
// The list of CPT and HCPCS codes for the various Covid vaccines can be found
// at the below website.
//
// https://www.cms.gov/medicare/medicare-part-b-drug-average-sales-price/covid-19-vaccines-and-monoclonal-antibodies

// I don't see productOrService in my sample data? Here's an example from the hl7 site:
// https://www.hl7.org/fhir/explanationofbenefit-example.json.html

// What we're looking for is an entry where some item in resource.item[] where "service[]" (XXX: or "product"? is that what "productOrService means?) has an entry "code" that matches one of:
// 91300: Pfizer
var VaxCodes map[string]bool = map[string]bool{
	"91300": true, // Pfizer-Biontech Covid-19 Vaccine
	"0001A": true, // Pfizer-Biontech Covid-19 Vaccine Administration – First Dose
	"0002A": true, // Pfizer-Biontech Covid-19 Vaccine Administration – Second Dose
	"91301": true, // Moderna Covid-19 Vaccine
	"0011A": true, // Moderna Covid-19 Vaccine Administration – First Dose
	"0012A": true, // Moderna Covid-19 Vaccine Administration – Second Dose
	"91302": true, // AstraZeneca Covid-19 Vaccine
	"0021A": true, // AstraZeneca Covid-19 Vaccine Administration – First Dose
	"0022A": true, // AstraZeneca Covid-19 Vaccine Administration – Second Dose
}

func findVaxes(e EOBResponse) ([]Vaccination, error) {
	var vaxes []Vaccination
	for _, entry := range e.Entries {
		for _, item := range entry.Resource.Items {
			for _, serviceCode := range item.ProductOrService.Coding {
				if _, ok := VaxCodes[serviceCode.Code]; ok {
					dt, err := time.Parse(time.RFC3339Nano, item.ServicedDate)
					if err != nil {
						return nil, err
					}
					vaxes = append(vaxes, Vaccination{
						Date:    dt,
						Code:    serviceCode.Code,
						Display: serviceCode.Display,
					})
				}
			}
		}
	}
	return vaxes, nil
}

func (c *Client) GetEOB(tok, fhirID string) (*EOBResponse, error) {
	var res EOBResponse
	// can we limit this to outputient or something like?
	// /v1/fhir/ExplanationOfBenefit?patient=123&type=carrier,dme,hha,hospice,inpatient,outpatient,snf
	err := get(fmt.Sprintf("%s/v1/fhir/ExplanationOfBenefit?patient=%s", c.BBURL, fhirID), tok, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) FindVaccionations(tok, fhirID string) ([]Vaccination, error) {
	var res EOBResponse

	// can we limit this to outputient or something like?
	// /v1/fhir/ExplanationOfBenefit?patient=123&type=carrier,dme,hha,hospice,inpatient,outpatient,snf
	err := get(fmt.Sprintf("%s/v1/fhir/ExplanationOfBenefit?patient=%s", c.BBURL, fhirID), tok, &res)
	if err != nil {
		return nil, err
	}
	vaxes, err := findVaxes(res)
	if err != nil {
		return nil, err
	}
	for next := res.Next(); next != ""; next = res.Next() {
		err := get(next, tok, &res)
		if err != nil {
			return nil, err
		}
		moreVaxes, err := findVaxes(res)
		if err != nil {
			return nil, err
		}
		vaxes = append(vaxes, moreVaxes...)
	}
	return vaxes, nil
}
