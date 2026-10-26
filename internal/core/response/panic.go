package response

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type panicError struct {
	message string
	stack   string
}

func (e *panicError) Error() string { return e.message }

// PanicHandler captures only function names and source positions, not argument
// values or other goroutines. Panics always become internal errors, even when
// their value is an otherwise public business/HTTP error.
func PanicHandler(_ fiber.Ctx, value any) error {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var stack strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&stack, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return &panicError{message: fmt.Sprint(value), stack: stack.String()}
}
