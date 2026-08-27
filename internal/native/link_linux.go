//go:build linux && amd64

package native

/*
#cgo CFLAGS: -I${SRCDIR}/artifacts/linux-amd64/include
#cgo LDFLAGS: -L${SRCDIR}/artifacts/linux-amd64/lib -ltirtc_media -Wl,-rpath,${SRCDIR}/artifacts/linux-amd64/lib
#include "bridge.h"
*/
import "C"
