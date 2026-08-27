package storage

import (
	"errors"
	"fmt"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

const (
	cloudStorageLogTag     = "Ti Cloud Storage"
	cloudStorageLogInfo    = uint32(1)
	cloudStorageLogWarning = uint32(2)
)

func logCloudStorageEvent(operation string) {
	_ = native.Log(cloudStorageLogInfo, cloudStorageLogTag, fmt.Sprintf("[%s]", operation))
}

func logCloudStorageResult(operation string, err error) {
	level := cloudStorageLogInfo
	code := int32(0)
	if err != nil {
		level = cloudStorageLogWarning
		var nativeErr *Error
		if errors.As(err, &nativeErr) {
			code = nativeErr.Code
		} else {
			code = -1
		}
	}
	_ = native.Log(level, cloudStorageLogTag, fmt.Sprintf("[%s] completed code=%d", operation, code))
}

func logCloudStorageState(operation string, state OutputState) {
	level := cloudStorageLogInfo
	if state == OutputFailed {
		level = cloudStorageLogWarning
	}
	_ = native.Log(level, cloudStorageLogTag, fmt.Sprintf("[%s] state=%d", operation, state))
}
