package utils

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
