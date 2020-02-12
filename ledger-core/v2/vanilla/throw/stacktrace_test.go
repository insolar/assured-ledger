package throw

import (
	"fmt"
	"testing"
)

func TestCaptureStack(t *testing.T) {
	// TODO proper tests
	fmt.Printf("%s\n==============\n", captureStack(0, false))
	fmt.Printf("%s\n==============\n", captureStack(1, false))
	fmt.Printf("%s\n==============\n", captureStack(1, true))
}

//func TestSystemPanic(t *testing.T) {
//	panic(IllegalState())
//}
