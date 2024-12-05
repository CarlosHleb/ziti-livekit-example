/*
	Copyright NetFoundry Inc.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package openziti

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/ziti/ziti/cmd/common"

	"github.com/michaelquigley/pfxlog"
	"github.com/openziti/sdk-golang/ziti/enroll"
	"github.com/pkg/errors"
)

// global state used by all subcommands are located here for easy discovery

const outpathDesc = "Output configuration file."
const jwtpathDesc = "Enrollment token (JWT file). Required"
const certDesc = "The certificate to present when establishing a connection."
const idnameDesc = "Names the identity. Ignored if not 3rd party auto enrollment"

const outFlag = "out"

// EnrollOptions contains the command line options
type EnrollOptions struct {
	common.CommonOptions
	RemoveJwt  bool
	KeyAlg     ziti.KeyAlgVar
	JwtPath    string
	JWTString  string
	OutputPath string
	KeyPath    string
	CertPath   string
	IdName     string
	CaOverride string
	Username   string
	Password   string
}

type EnrollAction struct {
	EnrollOptions
}

// Enroll identity with JWT path
// If JWTString is set in EnrollAction will enroll with that instead and return
// identitiy as a string
func (e *EnrollAction) Run() (string, error) {
	var tokenStr string

	if e.JWTString != "" {
		tokenStr = e.JWTString
	} else {
		if strings.TrimSpace(e.OutputPath) == "" {
			out, outErr := outPathFromJwt(e.JwtPath)
			if outErr != nil {
				return "", fmt.Errorf("could not set the output path: %s", outErr)
			}
			e.OutputPath = out
		}

		if e.JwtPath != "" {
			if _, err := os.Stat(e.JwtPath); os.IsNotExist(err) {
				return "", fmt.Errorf("the provided jwt file does not exist: %s", e.JwtPath)
			}
		}
		if strings.TrimSpace(e.OutputPath) == strings.TrimSpace(e.JwtPath) {
			return "", fmt.Errorf("the output path must not be the same as the jwt path")
		}
		tokenByte, _ := os.ReadFile(e.JwtPath)
		tokenStr = string(tokenByte)
	}

	pfxlog.Logger().Debugf("jwt to parse: %s", tokenStr)
	tkn, _, err := enroll.ParseToken(tokenStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %s", err.Error())
	}

	flags := enroll.EnrollmentFlags{
		CertFile:      e.CertPath,
		KeyFile:       e.KeyPath,
		KeyAlg:        e.KeyAlg,
		Token:         tkn,
		IDName:        e.IdName,
		AdditionalCAs: e.CaOverride,
		Username:      e.Username,
		Password:      e.Password,
		Verbose:       e.Verbose,
	}

	conf, err := enroll.Enroll(flags)
	if err != nil {
		return "", fmt.Errorf("failed to enroll: %v", err)
	}

	if e.JWTString != "" {
		output := new(bytes.Buffer)
		enc := json.NewEncoder(output)
		enc.SetEscapeHTML(false)
		encErr := enc.Encode(&conf)
		if encErr != nil {
			return "", fmt.Errorf("enrollment successfull, but failed to encode to json: %s", encErr)
		}
		return output.String(), nil
	} else {
		output, err := os.Create(e.OutputPath)
		if err != nil {
			return "", fmt.Errorf("failed to open file '%s': %s", e.OutputPath, err.Error())
		}
		defer func() { _ = output.Close() }()

		enc := json.NewEncoder(output)
		enc.SetEscapeHTML(false)
		encErr := enc.Encode(&conf)

		if err = os.Remove(e.JwtPath); err != nil {
			pfxlog.Logger().WithError(err).Warnf("unable to remove JWT file as requested: %v", e.JwtPath)
		}

		if encErr == nil {
			pfxlog.Logger().Infof("enrolled successfully. identity file written to: %s", e.OutputPath)
			return "", nil
		} else {
			return "", fmt.Errorf("enrollment successful but the identity file was not able to be written to: %s [%s]", e.OutputPath, encErr)
		}
	}
	return "", nil
}

func outPathFromJwt(jwt string) (string, error) {
	if strings.HasSuffix(jwt, ".jwt") {
		return jwt[:len(jwt)-len(".jwt")] + ".json", nil
	} else if strings.HasSuffix(jwt, ".json") {
		//ugh - so that makes things a bit uglier but ok fine. we'll return an error in this situation
		return "", errors.Errorf("unexpected configuration. cannot infer '%s' flag if the jwt file "+
			"ends in .json. rename jwt file or provide the '%s' flag", outFlag, outFlag)
	} else {
		//doesn't end with .jwt - so just slap a .json on the end and call it a day
		return jwt + ".json", nil
	}
}
