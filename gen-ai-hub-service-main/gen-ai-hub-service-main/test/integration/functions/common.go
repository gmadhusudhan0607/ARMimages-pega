//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"regexp"
	"time"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

const MockCreateModelExpectationReqWithPathTpl = `
	{
		"httpRequest" : {
			"method" : "POST",
			"path" : "%s",
			"headers": {
			  "test-id": "%s"
			}
		},
		"httpResponse" : {
			"statusCode": 200,
			"body" : "{}"
		}
	}`

const MockCreateModelExpectationReqWithPathAndBodyTpl = `
	{
		"httpRequest" : {
			"method" : "POST",
			"path" : "%s",
			"body": %s ,
			"headers": {
			  "test-id": "%s"
			}
		},
		"httpResponse" : {
			"statusCode": 200,
			"body" : "{}"
		}
	}`

const MockCreatePostExpectationReqWithPathAndResponseBodyTpl = `
	{
		"httpRequest" : {
			"method" : "POST",
			"path" : "%s"
		},
		"httpResponse" : {
			"statusCode": 200,
			"body": %s
		}
	}`

const MockCreatePostExpectationReqWithPathHeaderAndResponseBodyTpl = `
	{
		"httpRequest" : {
			"method" : "POST",
			"path" : "%s",
			"headers": {
			  "X-Amz-Target": ["%s"]
			}
		},
		"httpResponse" : {
			"statusCode": 200,
			"body": %s
		}
	}`

const MockCreateGetExpectationReqWithPathAndResponseBodyTpl = `
	{
		"httpRequest" : {
			"method" : "POST",
			"path" : "%s"
		},
		"httpResponse" : {
			"statusCode": 200,
			"body": "%s"
		}
	}`

const MockValidateExpectationReqTpl = `
	{
		"expectationId": {
			"id": "%s"
		},
		"times": {
			"atLeast": %d,
			"atMost": %d
		}
	}
`

// Create a new random number generator with a custom seed (e.g., current time)
var rndGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rndGenerator.Intn(len(letterRunes))]
	}
	return string(b)
}

func GetEnvOfDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GetBuddyByName(buddies []Buddy, name string) *Buddy {
	for i := range buddies {
		if buddies[i].Name == name {
			return &buddies[i]
		}
	}
	Expect(false).To(Equal(true), fmt.Sprintf("buddy '%s' not found", name))
	return nil
}

func GetModelByName(models []Model, name string) *Model {
	for i := range models {
		if models[i].Name == name {
			return &models[i]
		}
	}
	Expect(false).To(Equal(true), fmt.Sprintf("model '%s' not found", name))
	return nil
}

func GetGenAiInfraByName(secrets []GenAIInfraConfig, mappingName string) *GenAIInfraConfig {
	for i := range secrets {
		if secrets[i].ModelMapping == mappingName {
			return &secrets[i]
		}
	}
	Expect(false).To(Equal(true), fmt.Sprintf("model '%s' not found", mappingName))
	return nil
}

func ParseRegexParameters(str string, pattern string) map[string]string {
	result := make(map[string]string)
	r, err := regexp.Compile(pattern)
	Expect(err).To(BeNil(), fmt.Sprintf("failed to compile pattern '%s' ", pattern))
	m := r.FindStringSubmatch(str)
	if m != nil {
		for i, name := range r.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = m[i]
			}
		}
	}
	return result
}

func ReadFile(file string) string {
	fi, err := os.Stat(file)
	Expect(err).To(BeNil())
	size := fi.Size()
	Expect(size).NotTo(BeZero())

	f, err := os.Open(file)
	Expect(err).To(BeNil())
	defer f.Close()

	data := make([]byte, size)
	_, err = f.Read(data)
	Expect(err).To(BeNil())
	return string(data)
}

func LoadMappingFromFile(fileName string) (mappings Mappings, err error) {
	fContent := ReadFile(fileName)
	err = yaml.Unmarshal([]byte(fContent), &mappings)
	if err != nil {
		return mappings, fmt.Errorf("error when unmarshaling yaml file %s: %w", fileName, err)
	}
	return mappings, nil
}

func LoadMappingFromSecretsDir(dir ...string) ([]GenAIInfraConfig, error) {

	infraModels := make([]GenAIInfraConfig, 0)
	//list all files in the secret mount directory
	for _, d := range dir {
		files, err := os.ReadDir(d)
		if err != nil {
			return nil, fmt.Errorf("error when reading genai infra directory %s: %w", dir, err)
		}

		for _, f := range files {
			content, err := os.ReadFile(d + "/" + f.Name())
			if err != nil {
				return nil, fmt.Errorf("error when reading file %s: %w", f.Name(), err)
			}

			model := GenAIInfraConfig{}
			err = json.Unmarshal(content, &model)
			if err != nil {
				return nil, fmt.Errorf("error when unmarshaling json file %s: %w", f.Name(), err)
			}
			infraModels = append(infraModels, model)
		}
	}
	return infraModels, nil
}

func ToJSONString(v interface{}) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// GetFreePort finds a free TCP port on localhost and returns it.
// The port is freed immediately after discovery so it can be used by the caller.
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to find a free port: %w", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func isPortAccessible(host string, port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}
