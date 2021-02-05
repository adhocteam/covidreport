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
