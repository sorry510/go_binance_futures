package utils

import (
	"encoding/json"
	"math"
	"strconv"
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
	size_float64, _ := strconv.ParseFloat(size, 64)
	base_float64 := 1 / size_float64 // 放大多少倍
	return math.Round(number_float64 * base_float64) / base_float64
}

// 计算逻辑: ma(5)=(收盘价1+收盘价2+收盘价3+收盘价4+收盘价5)/5
func MaN(closePrice []float64, n int) float64 {
	allPrice := 0.0
	for i := 0; i < n; i++ {
		allPrice += closePrice[i]
	}
	return allPrice / float64(n)
}


// ma(n) 的 count 条数据
// @param 收盘价 klineClose 
// @param ma(n) n
// @param count 条数数据
func MaNList(closePrice []float64, n int, count int) ([]float64) {
	maNList := make([]float64, count)
	for i := 0; i < count; i++ {
		maNList[i] = MaN(closePrice[i:], n)
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