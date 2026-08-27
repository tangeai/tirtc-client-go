//go:build darwin && arm64

package native

/*
#cgo CFLAGS: -I${SRCDIR}/artifacts/darwin-arm64/include
#cgo LDFLAGS: -L${SRCDIR}/artifacts/darwin-arm64/lib -ltirtc_media -Wl,-rpath,${SRCDIR}/artifacts/darwin-arm64/lib
#include "bridge.h"
*/
import "C"
