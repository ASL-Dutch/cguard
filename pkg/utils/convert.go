package utils

import (
	"errors"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// StringToFloat64 safely converts string to float64
func StringToFloat64(s string) float64 {
	if s == "" {
		return 0.0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Warnf("Failed to convert string '%s' to float64, using 0.0: %v", s, err)
		return 0.0
	}
	return val
}

// StringToInt safely converts string to int
func StringToInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("string is empty")
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// GetStringValue safely gets string value from row with bounds checking
func GetStringValue(row []string, index int) string {
	if index >= len(row) {
		return ""
	}
	return row[index]
}

// GetFloatValue safely gets float value from row with bounds checking
func GetFloatValue(row []string, index int) float64 {
	if index >= len(row) {
		return 0.0
	}
	return StringToFloat64(row[index])
}

// GetIntValue safely gets int value from row with bounds checking
func GetIntValue(row []string, index int) int {
	if index >= len(row) {
		return 0
	}
	val, err := StringToInt(row[index])
	if err != nil {
		log.Warnf("Failed to convert string '%s' to int, using 0: %v", row[index], err)
		return 0
	}
	return val
}
