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

// 根据 tickSize 或 stepSize 获取精度
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

// 计算逻辑: ma(5)=(收盘价1+收盘价2+收盘价3+收盘价4+收盘价5)/5
func MaN(closePrice []float64, n int) float64 {
	allPrice := 0.0
	for i := 0; i < n; i++ {
		allPrice += closePrice[i]
	}
	return allPrice / float64(n)
}


// ma 的 n 条数据
func MaNList(closePrice []float64, n int, count int) (maNList []float64) {
	for i := 0; i < count; i++ {
		maNList = append(maNList, MaN(closePrice[i:], n))
	}
	return maNList
}

// 是否是一个降序数组
func IsDesc(arr []float64) bool {
	for i := 1; i < len(arr); i++ {
		if arr[i-1] <= arr[i] {
			return false
		}
	}
	return true
}

// 是否是一个升序数组
func IsAsc(arr []float64) bool {
	for i := 1; i < len(arr); i++ {
		if arr[i-1] >= arr[i] {
			return false
		}
	}
	return true
}