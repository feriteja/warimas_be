package user

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type CustomClaims struct {
	UserID   uint    `json:"user_id"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	SellerID *string `json:"seller_id,omitempty"`
	jwt.RegisteredClaims
}

var ErrEmailExists = errors.New("email already registered")

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateJWT(userID int, role, email string, sellerID *string) (string, error) {

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not set")
	}

	jwtKey := []byte(secret)
	claims := CustomClaims{
		UserID:   uint(userID),
		Email:    email,
		Role:     role,
		SellerID: sellerID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseJWT(tokenStr string) (*CustomClaims, error) {
	secret := (os.Getenv("JWT_SECRET"))
	if secret == "" {
		return nil, errors.New("JWT_SECRET is not set")
	}

	jwtKey := []byte(secret)

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
