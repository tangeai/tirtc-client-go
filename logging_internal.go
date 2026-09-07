package tirtc

import (
	"errors"
	"fmt"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

const (
	sdkLogTag     = "TiRTC"
	sdkLogInfo    = uint32(1)
	sdkLogWarning = uint32(2)
)

func logSDKEvent(operation string) {
	_ = native.Log(sdkLogInfo, sdkLogTag, fmt.Sprintf("[%s]", operation))
}

func logSDKResult(operation string, err error) {
	level := sdkLogInfo
	code := int32(0)
	if err != nil {
		level = sdkLogWarning
		var nativeErr *Error
		if errors.As(err, &nativeErr) {
			code = nativeErr.Code
		} else {
			code = -1
		}
	}
	_ = native.Log(level, sdkLogTag, fmt.Sprintf("[%s] completed code=%d", operation, code))
}

func logSDKState(operation string, state uint32, err error) {
	level := sdkLogInfo
	code := int32(0)
	if err != nil {
		level = sdkLogWarning
		var nativeErr *Error
		if errors.As(err, &nativeErr) {
			code = nativeErr.Code
		} else {
			code = -1
		}
	}
	_ = native.Log(level, sdkLogTag, fmt.Sprintf("[%s] state=%d code=%d", operation, state, code))
}
