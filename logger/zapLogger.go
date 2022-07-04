package logger

import "go.uber.org/zap"

type zapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(log *zap.Logger) *zapLogger {
	return &zapLogger{logger: log}

}
func (z zapLogger) Debug(str string) {
	z.logger.Debug(str)
}

func (z zapLogger) Info(str string) {
	z.logger.Info(str)
}

func (z zapLogger) Error(str string) {
	z.logger.Error(str)
}

func (z zapLogger) Fatal(str string) {
	z.logger.Fatal(str)
}
