package api

import (
	"errors"
	"strconv"
	"strings"
)

func parseCompanyPath(path string) (int64, string, string) {
	const prefix = "/api/v1/companies/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", "not_found"
	}

	rawPath := strings.TrimPrefix(path, prefix)
	if rawPath == "" {
		return 0, "", "not_found"
	}

	parts := strings.SplitN(rawPath, "/", 2)
	companyID, err := parsePositiveID(parts[0])
	if err != nil {
		return 0, "", "validation_error"
	}

	if len(parts) == 1 {
		return companyID, "", ""
	}
	if parts[1] == "" {
		return 0, "", "not_found"
	}

	return companyID, parts[1], ""
}

func parsePositiveID(rawID string) (int64, error) {
	value, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid id")
	}
	return value, nil
}

func ParseCompanyPath(path string) (int64, string, string) {
	return parseCompanyPath(path)
}

func ParsePositiveID(rawID string) (int64, error) {
	return parsePositiveID(rawID)
}
