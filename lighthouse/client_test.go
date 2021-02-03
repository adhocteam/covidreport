package lighthouse

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

	var pat PatientResponse
	if err := json.Unmarshal(jsonData, &pat); err != nil {
		t.Log(err)
		t.FailNow()
	}

	if len(pat.Names) == 0 {
		t.Errorf("empty names %#v", pat)
	}

	if pat.Names[0].Text != "Mr. Porfirio146 Schmeler639" {
		t.Errorf("unexpected patient %#v", pat.Names)
	}
}
