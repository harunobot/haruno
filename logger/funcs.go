package logger

// Field 设置logger的域
func Field(name string) LogInterface {
	return &loggerWithField{field: name, service: &Service}
}

// Success 成功log
func Success(text string) {
	Service.Success(text)
}

// Successf 格式化成功log
func Successf(format string, args ...interface{}) {
	Service.Successf(format, args...)
}

// Info 信息log
func Info(text string) {
	Service.Info(text)
}

// Infof 格式化信息log
func Infof(format string, args ...interface{}) {
	Service.Infof(format, args...)
}

// Error 错误log
func Error(a interface{}) {
	Service.Error(a)
}

// Errorf 格式化错误log
func Errorf(format string, args ...interface{}) {
	Service.Errorf(format, args...)
}
