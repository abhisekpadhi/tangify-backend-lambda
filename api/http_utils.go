package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

type apiResponseType struct {
	Success      func(body interface{}) events.LambdaFunctionURLResponse
	Error        func(statusCode int, message string) events.LambdaFunctionURLResponse
	BadRequest   func(message string) events.LambdaFunctionURLResponse
	Unauthorized func(message string) events.LambdaFunctionURLResponse
}

func jsonBody(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Println("error marshalling JSON: ", err)
		return "{\"error\":\"Internal server error\"}"
	}
	return string(b)
}

var ApiResponse = apiResponseType{
	Success: func(body interface{}) events.LambdaFunctionURLResponse {
		return events.LambdaFunctionURLResponse{
			StatusCode:      200,
			Headers:         map[string]string{"Content-Type": "application/json"},
			Body:            jsonBody(body),
			IsBase64Encoded: false,
		}
	},

	Error: func(statusCode int, message string) events.LambdaFunctionURLResponse {
		return events.LambdaFunctionURLResponse{
			StatusCode:      statusCode,
			Headers:         map[string]string{"Content-Type": "application/json"},
			Body:            jsonBody(map[string]string{"error": message}),
			IsBase64Encoded: false,
		}
	},

	BadRequest: func(message string) events.LambdaFunctionURLResponse {
		return events.LambdaFunctionURLResponse{
			StatusCode:      400,
			Headers:         map[string]string{"Content-Type": "application/json"},
			Body:            jsonBody(map[string]string{"error": message}),
			IsBase64Encoded: false,
		}
	},

	Unauthorized: func(message string) events.LambdaFunctionURLResponse {
		return events.LambdaFunctionURLResponse{
			StatusCode:      401,
			Headers:         map[string]string{"Content-Type": "application/json"},
			Body:            jsonBody(map[string]string{"error": message}),
			IsBase64Encoded: false,
		}
	},
}

