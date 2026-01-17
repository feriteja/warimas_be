package utils

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"warimas-be/internal/graph/model"

	"github.com/99designs/gqlgen/graphql"
)

var (
	nonAlnumRegex  = regexp.MustCompile(`[^a-z0-9]+`)
	multiDashRegex = regexp.MustCompile(`-+`)
)

func Slugify(input string, sellerID string) string {
	// Get the first part of sellerID
	sellerPrefix := strings.Split(sellerID, "-")[0]

	// Convert input to lowercase
	slug := strings.ToLower(input)

	// Trim whitespace
	slug = strings.TrimSpace(slug)

	// Replace non-alphanumeric characters with dash
	slug = nonAlnumRegex.ReplaceAllString(slug, "-")

	// Remove multiple dashes
	slug = multiDashRegex.ReplaceAllString(slug, "-")

	// Trim leading & trailing dashes
	slug = strings.Trim(slug, "-")

	// Prepend sellerPrefix to slug
	slug = sellerPrefix + "-" + slug

	return slug
}

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

func PtrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func PtrInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func PtrInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func HasAnyUpdateProductField(input model.UpdateProduct) bool {
	return input.Name != nil ||
		input.ImageURL != nil ||
		input.Description != nil ||
		input.CategoryID != nil ||
		input.SubcategoryID != nil ||
		input.Status != nil
}

func HasAnyVariantUpdateField(v *model.UpdateVariant) bool {
	return v.QuantityType != nil ||
		v.Name != nil ||
		v.Price != nil ||
		v.Stock != nil ||
		v.ImageURL != nil ||
		v.Description != nil
}

func HasAnyField(ctx context.Context, names ...string) bool {
	fields := graphql.CollectFieldsCtx(ctx, nil)

	for _, f := range fields {
		if slices.Contains(names, f.Name) {
			return true
		}
	}
	return false
}

func IsInternalRequest(ctx context.Context) bool {
	v := ctx.Value(internalRequestKey)
	if v == nil {
		return false
	}

	isInternal, ok := v.(bool)
	return ok && isInternal
}

func ExternalIDFromSession(prefix, sessionID string) string {
	h := sha1.Sum([]byte(sessionID))
	return fmt.Sprintf(
		"%s_%s",
		prefix,
		hex.EncodeToString(h[:6]), // short but safe
	)
}

func FormatIDR(amount int64) string {
	if amount == 0 {
		return "Rp 0"
	}

	s := strconv.FormatInt(amount, 10)
	n := len(s)

	var parts []string
	for n > 3 {
		parts = append([]string{s[n-3 : n]}, parts...)
		n -= 3
	}
	parts = append([]string{s[:n]}, parts...)

	return "Rp " + strings.Join(parts, ".")
}
