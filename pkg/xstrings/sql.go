package xstrings

import "strings"

func BuildInQuery(values []string) (string, []interface{}) {
	query := ""
	length := len(values)
	args := make([]interface{}, length)
	for i := 0; i < length; i++ {
		query += "?,"
		args[i] = values[i]
	}
	query = strings.TrimRight(query, ",")
	return query, args
}
