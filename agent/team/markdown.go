package team

import (
	"fmt"
	"strings"
)

func FormatMarkdown(result ResultV1) string {
	var out strings.Builder
	fmt.Fprintf(&out, "## %s 多智能体分析\n\n", result.Symbol)
	fmt.Fprintf(&out, "- **状态**：%s\n", result.Status)
	fmt.Fprintf(&out, "- **方向**：%s\n", result.Direction)
	fmt.Fprintf(&out, "- **置信度**：%.1f%%\n", result.Confidence*100)
	if strings.TrimSpace(result.Summary) != "" {
		fmt.Fprintf(&out, "\n### 综合结论\n\n%s\n", result.Summary)
	}
	writeMarkdownList(&out, "共识", result.Consensus)
	writeMarkdownList(&out, "分歧", result.Disagreements)
	writeMarkdownList(&out, "数据缺失", result.DataMissing)
	if len(result.Evidence) > 0 {
		out.WriteString("\n### Evidence\n\n")
		for _, item := range result.Evidence {
			fmt.Fprintf(&out, "- **%s** · `%s`：%s\n", item.Role, item.Source, item.Finding)
		}
	}
	return strings.TrimSpace(out.String())
}

func writeMarkdownList(out *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(out, "\n### %s\n\n", title)
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			fmt.Fprintf(out, "- %s\n", value)
		}
	}
}
