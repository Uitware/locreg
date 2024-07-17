package local_registry

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"reflect"
	"strings"
)

type DockerEngineLogString struct {
	Stream         string `json:"stream,omitempty"`
	Aux            string `json:"aux,omitempty"`
	Status         string `json:"status,omitempty"`
	ProgressDetail string `json:"progressDetail,omitempty"`
}

type ErrorLine struct {
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}

func GetNonNilFields(logString DockerEngineLogString) (result string) {
	// loop over all values in struct return only existing one and do this in sting format
	v := reflect.ValueOf(logString)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsZero() {
			result = field.String()
		}
	}
	return result
}

// PrintLog Get logs produced by docker engine and print them in fancy formating
// takes io.Reader as input and returns error if any occur in logs form docker engine
func PrintLog(rd io.Reader) error {
	var lastLine string
	var logString DockerEngineLogString

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		err := json.Unmarshal(scanner.Bytes(), &logString)
		if err != nil {
			continue // skip invalid json
		}
		if lg := strings.Trim(GetNonNilFields(logString), "\n"); lg != "" {
			log.Printf(lg)

		}
	}

	errLine := &ErrorLine{}
	if err := json.Unmarshal([]byte(lastLine), errLine); err == nil && errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
