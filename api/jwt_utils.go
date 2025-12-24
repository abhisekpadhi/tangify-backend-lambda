package main

import (
	"github.com/golang-jwt/jwt/v5"
)

type MyClaims struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

type JwtUtils struct {
	secret []byte
}

var jwtUtils *JwtUtils

func NewJwtUtils(secret string) *JwtUtils {
	if jwtUtils == nil {
		jwtUtils = &JwtUtils{
			secret: []byte(secret),
		}
		return jwtUtils
	}
	return jwtUtils
}

// ParseJWT parses a JWT token string using the given secret (HS256)
// and returns the custom claims
func (j *JwtUtils) ParseJWT(tokenString string) (*MyClaims, error) {
	claims := &MyClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
