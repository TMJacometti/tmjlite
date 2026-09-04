//go:build windows

package tmjlite

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ltmjlite_ffi
#cgo LDFLAGS: -lws2_32 -ladvapi32 -luserenv -lbcrypt -lntdll
*/
import "C"
