package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adhocteam/covidpassport/bluebutton"
	"github.com/adhocteam/covidpassport/lighthouse"
	"github.com/skip2/go-qrcode"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// qrCode accepts a string, encodes it into a PNG as a QR code, and returns the
// base64-encoded png
func genQrCode(in string) (string, error) {
	var qrCode []byte
	qrCode, err := qrcode.Encode(in, qrcode.Medium, 320)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(qrCode), nil
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	// We're just going to read the templates every time, to make development
	// easier. In the future, we might want to be more efficent with this
	t, err := template.ParseGlob("templates/*.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error %s", err.Error()), 500)
		return
	}

	err = t.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error %s", err.Error()), 500)
		return
	}
}

func logreq(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("path: %s", r.URL.Path)

		f(w, r)
	})
}

// CovidRecord represents a covid record server
type CovidRecord struct {
	Port     string
	BBClient bluebutton.Client
	VAClient lighthouse.Client
}

// Start a covid record server
func (s *CovidRecord) Start(cert, key string) {
	http.Handle("/callback", logreq(s.callbackHandler))
	http.Handle("/bbcallback", logreq(s.bbcallbackHandler))
	http.Handle("/error", logreq(serveError))
	http.Handle("/showCallback", logreq(staticCallback))
	http.Handle("/", logreq(s.defaultHandler))
	addr := fmt.Sprintf(":%s", s.Port)
	log.Printf("Starting covid record on %s", addr)
	if cert != "" {
		log.Fatal(http.ListenAndServeTLS(addr, cert, key, nil))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}

func (c *CovidRecord) defaultHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html", struct {
		VAAuthURL string
		BBAuthURL string
	}{
		BBAuthURL: c.BBClient.AuthURL(),
		VAAuthURL: c.VAClient.AuthURL("openid profile email launch/patient patient/Patient.read patient/Immunization.read"),
	})
}

func logHeaders(r *http.Request) {
	log.Printf("Headers:\n")
	for header, values := range r.Header {
		log.Printf("    %s: %s\n", header, values)
	}
}

// serveError is here so that we can test the error page when required
func serveError(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "error.html", fmt.Errorf("This is an example error"))
}

func mustParse(format, dt string) time.Time {
	tm, err := time.Parse(format, dt)
	if err != nil {
		panic(err)
	}
	return tm
}

// fakeVaccinations returns a given number of bluebutton vaccionations
func fakeVaccinations(nvax int) ([]bluebutton.Vaccination, *bluebutton.Patient) {
	vaxes := []bluebutton.Vaccination{}
	for i := 0; i < nvax; i++ {
		dt, err := time.Parse(time.RFC3339Nano, "2021-02-01T16:00:00.000+00:00")
		if err != nil {
			log.Printf(err.Error())
			panic(err)
		}

		// add 28 days for the second vax if present
		dt = dt.Add(time.Duration(i*28*24) * time.Hour)

		vaxes = append(vaxes, bluebutton.Vaccination{
			Date:     dt,
			Code:     "91300-0001A",
			Display:  fmt.Sprintf("COVID-19 Vaccination dose %d", i),
			Location: "Northshore Clinic - Skokie",
			Lot:      "1S892X78-B",
		})
	}
	patient := &bluebutton.Patient{
		BirthDate: bluebutton.YearMonthDay{Time: mustParse("2006-01-02", "1999-06-01")},
		Name: []bluebutton.PatientName{{
			Family: "Esposito",
			Given:  []string{"Joseph"},
		}},
	}

	return vaxes, patient
}

func staticCallback(w http.ResponseWriter, r *http.Request) {
	var nvax int
	if svax, ok := r.URL.Query()["vax"]; ok {
		var err error
		nvax, err = strconv.Atoi(svax[0])
		if err != nil {
			log.Printf(err.Error())
			nvax = 1
		}
	} else {
		nvax = 1
	}

	vaxes, patient := fakeVaccinations(nvax)

	fullToken := &bluebutton.FullToken{
		AccessToken: "123545",
	}

	dosesRemaining := fmt.Sprintf(`<span class="font-sans-lg">%d</span> doses remaining`, 2-len(vaxes))

	vaxComplete := len(vaxes) > 1

	name := fmt.Sprintf("%s %s", strings.Join(patient.Name[0].Given, " "), patient.Name[0].Family)

	var qrCode string
	var err error
	if vaxComplete {
		qrCode, err = genQrCode("✓")
	} else {
		qrCode, err = genQrCode("❌")
	}
	if err != nil {
		log.Printf("error getting eob: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	data := struct {
		User           *bluebutton.UserInfo
		Vaccinations   []bluebutton.Vaccination
		VaxComplete    bool
		FullToken      *bluebutton.FullToken
		Patient        *bluebutton.Patient
		QrCodePng      string
		DosesRemaining template.HTML
		Name           string
	}{
		// TODO: do I have access to the DOB?
		User: &bluebutton.UserInfo{
			GivenName:  "Benjamin",
			FamilyName: "Esposito",
			Name:       "Benjamin Esposito",
			Email:      "bennyespo@hotmail.com",
			FhirID:     "-209838199",
		},
		Vaccinations:   vaxes,
		VaxComplete:    vaxComplete,
		FullToken:      fullToken,
		Patient:        patient,
		QrCodePng:      qrCode,
		DosesRemaining: template.HTML(dosesRemaining),
		Name:           name,
	}

	renderTemplate(w, "callback.html", data)
}

// https://bluebutton.cms.gov/developers/#client-application-flow
func (c *CovidRecord) bbcallbackHandler(w http.ResponseWriter, r *http.Request) {
	logHeaders(r)

	// pull the token out of the callback parameters
	// XXX: check state param?
	codes := r.URL.Query()["code"]
	if len(codes) == 0 {
		renderTemplate(w, "error.html", fmt.Errorf("Unable to find a token in response %#v", r.URL))
		return
	}

	callbackToken := codes[0]

	fullToken, err := c.BBClient.GetFullToken(callbackToken)
	if err != nil {
		log.Printf("error getting full token: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	user, err := c.BBClient.GetUserInfo(fullToken.AccessToken)
	log.Printf("%#v", user)
	if err != nil {
		log.Printf("error getting user: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	patient, err := c.BBClient.GetPatient(user.FhirID, fullToken.AccessToken)
	log.Printf("%#v", patient)
	if err != nil {
		log.Printf("error getting patient: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	// For demo purposes, there are two special users. BBUser00000 is assumed
	// to have completed both their courses of vaccination, and BBUser11111
	// only one
	var vaxes []bluebutton.Vaccination
	if user.FhirID == "-19990000000001" {
		vaxes, _ = fakeVaccinations(2)
	} else if user.FhirID == "-20000000001112" {
		vaxes, _ = fakeVaccinations(1)
	} else {
		// XXX: in real life we should probably show the user a "you have
		// successfully loaded" page, show a spinner, and say "checking vaccination
		// records..." or something alike
		vaxes, err = c.BBClient.FindVaccionations(fullToken.AccessToken, user.FhirID)
		if err != nil {
			log.Printf("error getting eob: %s", err)
			renderTemplate(w, "error.html", err)
			return
		}
	}

	log.Printf("vaxes: %v", vaxes)

	dosesRemaining := fmt.Sprintf(`<span class="font-sans-lg">%d</span> doses remaining`, 2-len(vaxes))
	vaxComplete := len(vaxes) > 1

	var qrCode string
	if vaxComplete {
		qrCode, err = genQrCode("✓")
	} else {
		qrCode, err = genQrCode("❌")
	}
	if err != nil {
		log.Printf("error getting eob: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	// this is a tricky one. This will do for now, but would bear a lot more
	// thought in a real app
	name := fmt.Sprintf("%s %s", strings.Join(patient.Name[0].Given, " "), patient.Name[0].Family)

	data := struct {
		Vaccinations   []bluebutton.Vaccination
		Patient        *bluebutton.Patient
		QrCodePng      string
		DosesRemaining template.HTML
		Name           string
	}{
		Vaccinations:   vaxes,
		Patient:        patient,
		QrCodePng:      qrCode,
		DosesRemaining: template.HTML(dosesRemaining),
		Name:           name,
	}
	renderTemplate(w, "callback.html", data)
}

// https://developer.va.gov/explore/health/docs/authorization
// https://github.com/department-of-veterans-affairs/vets-api-clients/blob/master/test_accounts.md
func (c *CovidRecord) callbackHandler(w http.ResponseWriter, r *http.Request) {
	logHeaders(r)

	// pull the token out of the callback parameters
	// XXX: check state param?
	codes := r.URL.Query()["code"]
	if len(codes) == 0 {
		renderTemplate(w, "error.html", fmt.Errorf("Unable to find a token in response %#v", r.URL))
		return
	}

	callbackToken := codes[0]

	states := r.URL.Query()["state"]
	if len(states) == 0 {
		renderTemplate(w, "error.html", fmt.Errorf("Unable to find a token in response %#v", r.URL))
		return
	}
	state := states[0]

	fullToken, err := c.VAClient.GetFullToken(callbackToken, state)
	if err != nil {
		log.Printf("error getting full token: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	patient, err := c.VAClient.GetPatient(fullToken.AccessToken, fullToken.PatientID)
	log.Printf("%#v", patient)
	if err != nil {
		log.Printf("error getting user: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	vaxes, err := c.VAClient.GetVaccinations(fullToken.AccessToken, fullToken.PatientID)
	log.Printf("%#v", vaxes)
	if err != nil {
		log.Printf("error getting user: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	dosesRemaining := fmt.Sprintf(`<span class="font-sans-lg">%d</span> doses remaining`, 2-len(vaxes))
	vaxComplete := len(vaxes) > 1

	var qrCode string
	if vaxComplete {
		qrCode, err = genQrCode("✓")
	} else {
		qrCode, err = genQrCode("❌")
	}
	if err != nil {
		log.Printf("error getting eob: %s", err)
		renderTemplate(w, "error.html", err)
		return
	}

	data := struct {
		Vaccinations   []lighthouse.Vaccination
		Patient        *lighthouse.Patient
		QrCodePng      string
		DosesRemaining template.HTML
		Name           string
	}{
		Vaccinations:   vaxes,
		Patient:        patient,
		QrCodePng:      qrCode,
		DosesRemaining: template.HTML(dosesRemaining),
		Name:           patient.Name,
	}
	renderTemplate(w, "callback.html", data)
}

func (c *CovidRecord) String() string {
	return fmt.Sprintf(`Covid Record
	port: %s
	BBclient: %s
	VAclient: %s`, c.Port, c.BBClient.String(), c.VAClient.String())
}

func mustEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("Unable to find key %s", key))
	}
	return val
}

func env(key, adefault string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return adefault
	}
	return val
}

// secret returns the value of a given secret key for the current project
func secret(key string) string {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", mustEnv("PROJECT_ID"), key)

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		panic(err)
	}

	return string(result.Payload.Data)
}

func main() {
	// a local VA_CLIENT_SECRET overrides the google app secret, useful for
	// local testing
	vaClientSecret := env("VA_CLIENT_SECRET", "")
	if vaClientSecret == "" {
		vaClientSecret = secret("VA_CLIENT_SECRET")
	}
	vaClient := lighthouse.Client{
		ClientID:     mustEnv("VA_CLIENT_ID"),
		ClientSecret: vaClientSecret,
		URL:          mustEnv("VA_URL"),
		FhirURL:      mustEnv("VA_FHIR_URL"),
		CallbackURL:  mustEnv("VA_REDIRECT_URL"),
	}

	bbClient := bluebutton.Client{
		BBClientID:     mustEnv("BB_CLIENT_ID"),
		BBClientSecret: secret("BB_CLIENT_SECRET"),
		BBURL:          mustEnv("BB_URL"),
		CallbackURL:    mustEnv("BB_REDIRECT_URL"),
	}

	cert := os.Getenv("SSL_CERT")
	key := os.Getenv("SSL_KEY")
	covidRecordPort := env("COVID_RECORD_PORT", "6655")

	server := CovidRecord{
		Port:     covidRecordPort,
		VAClient: vaClient,
		BBClient: bbClient,
	}

	log.Printf("%s", server.String())
	server.Start(cert, key)
}
