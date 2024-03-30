package utils

import (
	"encoding/json"
	"math"
)

func ResJson(code int, data map[string]interface{}, msg ...string) interface{} {
	var message string
	if (code == 200) {
		message = "success"
	}
	if len(msg) != 0 {
		message = msg[0]
	}
	res := map[string]interface{} {
		"code": code,
		"msg": message,
		"data": data,
	}
	return res
}


func ToJson(data interface{}) string {
    b, err := json.Marshal(data)
    if err != nil {
        return ""
    }
    return string(b)
}

// e.i: (number "0.323443", size "0.01") => 0.32
func GetTradePrecision(number_float64 float64, size string) float64 {
	if size == "0.1" {
		return math.Round(number_float64*10) / 10
	} else if size == "0.01" {
		return math.Round(number_float64*100) / 100
	} else if size == "0.001" {
		return math.Round(number_float64*1000) / 1000
	} else if size == "0.0001" {
		return math.Round(number_float64*10000) / 10000
	} else if size == "0.00001" {
		return math.Round(number_float64*100000) / 100000
	} else if size == "0.000001" {
		return math.Round(number_float64*1000000) / 1000000
	} else {
		return math.Round(number_float64)
	}
}