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
