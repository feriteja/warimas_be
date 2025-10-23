package utils

import (
	"fmt"
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
