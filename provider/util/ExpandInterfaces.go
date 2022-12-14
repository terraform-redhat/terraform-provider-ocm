package util

func ExpandStringMap(m map[string]interface{}) map[string]string {
	stringMap := make(map[string]string, len(m))
	for k, v := range m {
		stringMap[k] = v.(string)
	}
	return stringMap
}

func ExpandStringValueList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, v.(string))
		}
	}
	return vs
}
