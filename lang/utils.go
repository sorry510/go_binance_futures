package lang

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

var langTextMap, _ = ReadLangJsonFile("") // 全局加载

func GetLanguage() string {
	var language, _ = config.String("language") // 通知语言
	return language
}

func ReadLangJsonFile(filePath string) (langTextMap map[string]interface{}, err error) {
	if filePath == "" {
		lang := GetLanguage()
		logs.Info("use lang:", lang)
		filePath = fmt.Sprintf("./lang/config/%s.json", lang)
	}
	jsonFile, err := os.Open(filePath)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return langTextMap, err
    }
    defer jsonFile.Close()

    // 读取文件内容
    byteValue, _ := io.ReadAll(jsonFile)

	if err := json.Unmarshal(byteValue, &langTextMap); err != nil {
        fmt.Println("Error unmarshalling JSON:", err)
        return langTextMap, err
    }
	return langTextMap, nil
}

func Lang(key string) (text string) {
	defer func() {
        if r := recover(); r != nil {
            text = key
        }
    }()
	keys := strings.Split(key, ".")
	
	tempText := langTextMap
	for index, k := range keys {
		if tempText[k] == nil {
			return key
		}
		if index < len(keys) - 1 {
			tempText = tempText[k].(map[string]interface{})
		} else {
			text = fmt.Sprintf("%s", tempText[k])
		}
	}
	return text
}

func LangMatch(text string) string {
    result := regexp.
		MustCompile(`\{([^}]+)\}`).
		ReplaceAllStringFunc(text, func (match string) string {
			content := match[1 : len(match)-1]
			return Lang(content)
		})
	return result
}

// 蛇形转驼峰，首字母大写
func ToCamelCase(snakeStr string) string {
    components := strings.Split(snakeStr, "_")
    for i := 0; i < len(components); i++ {
		if components[i] != "" {
			components[i] = strings.ToUpper(components[i][:1]) + components[i][1:]
		}
    }
    return strings.Join(components, "")
}

