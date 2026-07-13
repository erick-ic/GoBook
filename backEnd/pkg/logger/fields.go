package logger

func String(key string, value string) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}
