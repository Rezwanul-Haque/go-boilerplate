package ports

type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, err error, fields ...any)
	Debug(msg string, fields ...any)
	Warn(msg string, fields ...any)
}
