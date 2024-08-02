package azure

import (
	"encoding/json"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"io"
	"log"
)

// AzureError represents the structure of an error response from Azure
type AzureError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// DetailedAzureError represents a detailed error response structure from Azure
type DetailedAzureError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// ParseAndLogAzureError parses the JSON error response and logs the code and message
func ParseAndLogAzureError(responseBody []byte) {
	var azureError AzureError
	var detailedAzureError DetailedAzureError

	if err := json.Unmarshal(responseBody, &azureError); err == nil && azureError.Error.Code != "" {
		log.Printf("ERROR CODE: %s ERROR MESSAGE: %s\n", azureError.Error.Code, azureError.Error.Message)
		return
	}

	if err := json.Unmarshal(responseBody, &detailedAzureError); err == nil && detailedAzureError.Code != "" {
		log.Printf("ERROR CODE: %s ERROR MESSAGE: %s\n", detailedAzureError.Code, detailedAzureError.Message)
		return
	}

	log.Printf("Unrecognized error format: %s\n", string(responseBody))
}

func handleAzureError(err error) {
	var httpErr *azcore.ResponseError
	if errors.As(err, &httpErr) {
		responseBody, readErr := io.ReadAll(httpErr.RawResponse.Body)
		if readErr == nil {
			ParseAndLogAzureError(responseBody)
		} else {
			log.Printf("❌ Error reading response body: %v", readErr)
		}
	} else {
		log.Printf("❌ Error: %v", err)
	}
}
