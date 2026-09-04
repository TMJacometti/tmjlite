//go:build !windows

package tmjlite

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ltmjlite_ffi -Wl,-rpath,${SRCDIR}/lib
*/
import "C"
