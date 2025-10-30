package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func StrPtr(s string) *string {
	return &s
}

func ToUint(id string) (uint, error) {
	n, err := strconv.ParseUint(id, 10, 64)
	return uint(n), err
}

func ParseUint(s string) uint {
	var id uint
	fmt.Sscan(s, &id)
	return id
}

func WriteJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
