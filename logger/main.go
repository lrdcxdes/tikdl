package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	allowLogFile bool
	atomicLevel  = zap.NewAtomicLevelAt(zap.InfoLevel)
)

func Init() {
	simpleTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     simpleTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		atomicLevel,
	)
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.FatalLevel),
	)
	zap.ReplaceGlobals(logger)
}

func SetLevel(level string) {
	parsedLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		log.Panicf("failed to parse log level %s: %v", level, err)
	}
	atomicLevel.SetLevel(parsedLevel.Level())
}

func SetLogFile(value bool) {
	allowLogFile = value
}

func Sync() error {
	return zap.L().Sync()
}

func WriteFile(name string, content any) {
	if !allowLogFile {
		return
	}

	baseName := strings.TrimSuffix(name, filepath.Ext(name))

	var rawData []byte
	var isJSON bool
	var err error

	switch v := content.(type) {
	case *http.Response:
		if v.Body != nil {
			if seeker, ok := v.Body.(io.Seeker); ok {
				currentPos, _ := seeker.Seek(0, io.SeekCurrent)
				rawData, _ = io.ReadAll(v.Body)
				seeker.Seek(currentPos, io.SeekStart)
			} else {
				bodyBytes, err := io.ReadAll(v.Body)
				if err != nil {
					zap.S().Errorf("failed to read HTTP response body: %v", err)
					return
				}

				rawData = bodyBytes
				v.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
	case json.RawMessage:
		rawData = v
		isJSON = true
	case []byte:
		rawData = v
	case string:
		rawData = []byte(v)
	case fmt.Stringer:
		rawData = []byte(v.String())
	default:
		jsonData, err := sonic.ConfigFastest.Marshal(content)
		if err == nil {
			rawData = jsonData
			isJSON = true
		} else {
			rawData = fmt.Appendf(nil, "%v", v)
		}
	}

	if !isJSON && len(rawData) > 0 {
		if rawData[0] == '{' || rawData[0] == '[' {
			var dummy any
			if json.Unmarshal(rawData, &dummy) == nil {
				isJSON = true
			}
		}
	}

	var filePath string
	if isJSON {
		var prettyJSON []byte

		var jsonObj any
		if err := sonic.ConfigFastest.Unmarshal(rawData, &jsonObj); err == nil {
			prettyJSON, err = sonic.ConfigFastest.MarshalIndent(jsonObj, "", "  ")
			if err == nil {
				rawData = prettyJSON
			}
		}

		// fallback to standard json package if sonic fails
		if prettyJSON == nil {
			var indented bytes.Buffer
			if json.Indent(&indented, rawData, "", "  ") == nil {
				rawData = indented.Bytes()
			}
		}

		filePath = baseName + ".json"
	} else {
		filePath = baseName + ".txt"
	}

	err = os.WriteFile(filePath, rawData, 0644)
	if err != nil {
		zap.S().Errorf("failed to write file %s: %v", filePath, err)
		return
	}

	zap.S().Info("saved file " + filePath)
}
