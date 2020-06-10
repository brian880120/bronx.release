package common

import "strings"

func ParseLabel(labels []string) string {
	var sb strings.Builder
	for _, str := range labels {
		sb.WriteString("`" + str + "` ")
	}
	return sb.String()
}

func GetSubstringAfter(value string, target string) string {
	pos := strings.LastIndex(value, target)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(target)
	if adjustedPos >= len(value) {
		return ""
	}

	result := value[adjustedPos:]
	return strings.TrimSpace(strings.TrimPrefix(result, " - "))
}
