package bluebutton

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestParseEOB(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/outpatient.json")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	var eob EOBResponse
	if err := json.Unmarshal(jsonData, &eob); err != nil {
		t.Log(err)
		t.FailNow()
	}

	if eob.Total != 4 {
		t.Errorf("expected to find 4 entries, got %d", eob.Total)
	}
}
