package bluebutton

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestParsePatient(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/patient.json")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	var pat Patient
	if err := json.Unmarshal(jsonData, &pat); err != nil {
		t.Log(err)
		t.FailNow()
	}

	if pat.Name[0].Family != "Doe" || pat.Address[0].PostalCode != "99999" {
		t.Errorf("unexpected patient %#v", pat)
	}
}
