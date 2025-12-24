package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func getJwtClaims(jwtToken string, jwtSecret string) (*MyClaims, error) {
	jwtUtils := NewJwtUtils(jwtSecret)
	claims, err := jwtUtils.ParseJWT(jwtToken)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

var whitelistedRoutes = []string{
	"/api/v1/auth/login",
}

type AppContext struct {
	JWTClaims *MyClaims
}

func NewAppContext(claims *MyClaims) *AppContext {
	return &AppContext{
		JWTClaims: claims,
	}
}

func doJwtAuth(request events.LambdaFunctionURLRequest, jwtSecret string, appContext *AppContext) error {
	token := strings.TrimPrefix(request.Headers["authorization"], "Bearer ")
	if token == "" {
		fmt.Println("missing JWT")
		return ErrMissingJWT
	}
	claims, err := getJwtClaims(token, jwtSecret)
	if err != nil {
		fmt.Println("error parsing JWT: ", err)
		return ErrInvalidJWT
	}

	appContext.JWTClaims = claims

	return nil
}

func handler(request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	awsUtils := NewAwsUtils()
	route := request.RawPath
	fmt.Println("route: ", route)
	appContext := NewAppContext(nil)

	// if route not in whitelist, do jwt auth
	if !slices.Contains(whitelistedRoutes, route) {
		jwtSecret, err := awsUtils.GetSSMParameter("tangify.jwt.secret")
		if err != nil {
			fmt.Println("error getting JWT secret: ", err)
			return ApiResponse.Error(http.StatusInternalServerError, "Server error: Failed to get JWT secret"), nil
		}

		err = doJwtAuth(request, jwtSecret, appContext)
		if err != nil {
			return ApiResponse.Unauthorized(fmt.Sprintf("Unauthorized: %v", err)), nil
		}
	}

	fmt.Println("appContext: ", appContext)

	// resp, err := http.Get(DefaultHTTPGetAddress)
	// if err != nil {
	// 	return ApiResponse.Error(500, "Failed to get IP"), nil
	// }

	// if resp.StatusCode != 200 {
	// 	return ApiResponse.Error(500, "Failed to get IP"), nil
	// }

	// ip, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return ApiResponse.Error(500, "Failed to read IP"), nil
	// }

	// if len(ip) == 0 {
	// 	return ApiResponse.BadRequest("Invalid IP"), nil
	// }

	return ApiResponse.Success(map[string]string{"message": "Hello, World!"}), nil
}

func main() {
	lambda.Start(handler)
}
