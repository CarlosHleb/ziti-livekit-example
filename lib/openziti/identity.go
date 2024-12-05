package openziti

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	RoleAdmin               = "admin"
	RoleEnroller            = "enroller"
	RoleDevice              = "device"
	RoleDevicePendingEnroll = "device-pending-enroll"
	RoleInactive            = "inactive"
)

type zitiCreateIdentityResp struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func GetRoleAttributes(role string) ([]string, error) {
	switch role {
	case RoleAdmin:
		return []string{
			os.Getenv("ZITI_SERVICE_API") + ".dial",
			os.Getenv("ZITI_SERVICE_FRONTEND") + ".dial",
			os.Getenv("ZITI_SERVICE_LIVEKIT_RTC") + ".dial",
			os.Getenv("ZITI_SERVICE_LIVEKIT") + ".dial",
			os.Getenv("ZITI_SERVICE_TURN") + ".dial",
			os.Getenv("ZITI_SERVICE_ZAC") + ".dial",
		}, nil
	case RoleEnroller:
		return []string{
			os.Getenv("ZITI_SERVICE_DMZ") + ".dial",
		}, nil
	case RoleDevicePendingEnroll:
		return []string{
			os.Getenv("ZITI_SERVICE_DMZ") + ".dial",
		}, nil
	case RoleDevice:
		return []string{
			os.Getenv("ZITI_SERVICE_LIVEKIT_RTC") + ".dial",
			os.Getenv("ZITI_SERVICE_LIVEKIT") + ".dial",
			os.Getenv("ZITI_SERVICE_NATS") + ".dial",
			os.Getenv("ZITI_SERVICE_TURN") + ".dial",
		}, nil
	case RoleInactive:
		return []string{}, nil
	default:
		return nil, errors.New("invalid role")
	}
}

// Create openziti identity with enrollment ott(jwt)
// return id, jwt, err
func CreateIdentity(name string, role string) (string, string, error) {
	// Set role attributes
	roleAttributes, err := GetRoleAttributes(role)
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	isAdmin := false
	if role == RoleAdmin {
		isAdmin = true
	}

	// The data you want to send in JSON format
	jsonData := map[string]interface{}{
		"isAdmin": isAdmin,
		"name":    name,
		"type":    "User",
		"enrollment": map[string]interface{}{
			"ott": true,
		},
		"roleAttributes": roleAttributes,
	}

	// Marshal the JSON data
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		log.Print(err)
		return "", "", err
	}

	// Create a new HTTP POST request
	url := os.Getenv("ZITI_CTRL_URL") + "/edge/management/v1/identities"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("zt-session", SessionToken)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return "", "", err
	}

	if resp.StatusCode != http.StatusCreated {
		log.Print(string(body))
		log.Print(resp.StatusCode)
		return "", "", errors.New("failed to create openzity identity")
	}

	// Get jwt
	r := zitiCreateIdentityResp{}
	if err = json.Unmarshal(body, &r); err != nil {
		log.Print(err)
		return "", "", err
	}

	iden, err := GetIdentity(r.Data.ID)
	if err != nil {
		log.Print(err)
		return "", "", err
	}

	return r.Data.ID, iden.Data.Enrollment.OTT.JWT, nil
}

// Update openziti identity
// return err
func UpdateIdentity(id string, name string, role string) error {
	// Set role attributes
	roleAttributes, err := GetRoleAttributes(role)
	if err != nil {
		log.Print(err)
		return err
	}
	isAdmin := false
	if role == RoleAdmin {
		isAdmin = true
	}

	// The data you want to send in JSON format
	jsonData := map[string]interface{}{
		"isAdmin": isAdmin,
		"name":    name,
		"type":    "User",
		"enrollment": map[string]interface{}{
			"ott": true,
		},
		"roleAttributes": roleAttributes,
	}

	// Marshal the JSON data
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		log.Print(err)
		return err
	}

	// Create a new HTTP POST request
	url := os.Getenv("ZITI_CTRL_URL") + "/edge/management/v1/identities/" + id
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Print(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("zt-session", SessionToken)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		log.Print(string(body))
		log.Print(resp.StatusCode)
		return errors.New("failed to update openzity identity")
	}

	return nil
}

type zitiIdentity struct {
	Data struct {
		IsAdmin    bool `json:"isAdmin"`
		Enrollment struct {
			OTT struct {
				JWT string `json:"jwt"`
			} `json:"ott"`
		} `json:"enrollment"`
		RoleAttributes []string `json:"roleAttributes"`
	} `json:"data"`
}

// Get openziti identity info
func GetIdentity(id string) (iden zitiIdentity, err error) {
	// Make the HTTPS GET request
	url := os.Getenv("ZITI_CTRL_URL") + "/edge/management/v1/identities/" + id
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
		return iden, err
	}

	iden = zitiIdentity{}
	if err = json.Unmarshal(body, &iden); err != nil {
		log.Print(err)
		return iden, err
	}

	return iden, nil
}

// Delete openziti identity
func DeleteIdentity(id string) (err error) {
	url := os.Getenv("ZITI_CTRL_URL") + "/edge/management/v1/identities/" + id
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Print(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("zt-session", SessionToken)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
			return err
		}
		log.Print(string(body))

		log.Printf("Error deleting identity, id: %s, error code: %d", id, resp.StatusCode)
		return errors.New("openziti delete failed")
	}

	return nil
}
