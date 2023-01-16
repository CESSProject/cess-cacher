package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Uld        *zap.Logger
	LogPath    = "./log"
	uldLogPath = LogPath + "/uld.log"
)

func InitLogger() {
	f, err := os.Stat(LogPath)
	if err != nil {
		err = os.MkdirAll(LogPath, 0755)
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}
	} else if !f.IsDir() {
		err = os.RemoveAll(LogPath)
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}
		err = os.MkdirAll(LogPath, os.ModeDir)
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}
	}
	if _, err = os.Stat(uldLogPath); err != nil {
		f, err := os.Create(uldLogPath)
		if err != nil {
			log.Fatalf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		}
		f.Close()
	}
	initUldLogger()
}

// out log
func initUldLogger() {
	hook := lumberjack.Logger{
		Filename:   uldLogPath,
		MaxSize:    10,  //MB
		MaxAge:     365, //Day
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.InfoLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	Uld = zap.New(core, caller, development)
	Uld.Sugar().Errorf("The service has started and created a log file in the %v", uldLogPath)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()))
}
