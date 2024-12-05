package openziti

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/openziti/identity"
	edge_apis "github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/ziti/ziti/cmd/common"
)

var SessionToken string

func EnrollIfNeeded(zitiIDPath string) error {
	// Enroll openziti's apis identity if jwt exists
	_, err := os.Stat(zitiIDPath + ".jwt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
		} else {
			// File may exist but there's another error (e.g., permission denied)
			log.Print(err)
			return err
		}
	} else {
		// Enroll the openziti identity
		p := common.NewOptionsProvider(os.Stdout, os.Stdout)
		action := &EnrollAction{
			EnrollOptions: EnrollOptions{
				CommonOptions: p(),
				JwtPath:       zitiIDPath + ".jwt",
				KeyAlg:        "RSA",
			},
		}
		log.Printf("Enrolled identity: %s", zitiIDPath)
		_, err = action.Run()
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

// Using .json identity file, authenticates to openziti controller and gets api session
// stores it as global var
func CreateApiSession(ctrlUrl string, zitiIDPath string) error {
	// Read the JSON file
	jsonFile, err := os.ReadFile(zitiIDPath + ".json")
	if err != nil {
		log.Print(err)
		return err
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		log.Print(err)
		return err
	}

	// Create a new map[interface{}]interface{}
	convertedMap := make(map[interface{}]interface{})

	// Copy each key-value pair from the original map to the new map
	for key, value := range jsonData["id"].(map[string]interface{}) {
		convertedMap[key] = value
	}

	// Get api session from openziti
	urls := make([]*url.URL, 1)
	apiUrl, _ := url.Parse(ctrlUrl + "/edge/management/v1")
	urls[0] = apiUrl

	// Create Identitiy credentials from .json identity
	conf, err := identity.NewConfigFromMap(convertedMap)
	if err != nil {
		log.Print(err)
		return err
	}
	credentials := edge_apis.NewIdentityCredentialsFromConfig(*conf)

	// Authenticate and get token
	var configTypes []string
	managementClient := edge_apis.NewManagementApiClient(urls, credentials.GetCaPool(), func(ch chan string) {})
	apiSesionDetial, err := managementClient.Authenticate(credentials, configTypes)
	if err != nil {
		log.Print(err)
		return err
	}
	_, token := apiSesionDetial.GetAccessHeader()
	SessionToken = token

	if os.Getenv("DEV_ENV") == "true" {
		log.Printf("Ziti session token: %s", SessionToken)
	} else {
		log.Print("Created ziti api session")
	}
	return nil
}

// Pings openziti controller every X seconds to keep the api session alive
func KeepAlive(ctrlUrl string) {
	ticker := time.NewTicker(time.Minute * 10)

	for range ticker.C {
		// Make the HTTPS GET request
		url := ctrlUrl + "/edge/management/v1/"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Print(err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("zt-session", SessionToken)

		// Perform the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Print(err)
			return
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
			return
		}

		// Print the response body
		log.Printf("Performed keep alive for openziti session response: %s", body)
	}
}
