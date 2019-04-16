package sys

import (
	"os"

	"github.com/haruno-bot/haruno/logger"
	"golang.org/x/sys/windows"
)

// FixConsole 修复console的系统差异
func FixConsole() {
	in := windows.Handle(os.Stdin.Fd())
	var inMode uint32
	if err := windows.GetConsoleMode(in, &inMode); err == nil {
		var mode uint32
		// Disable these modes
		mode &^= windows.ENABLE_QUICK_EDIT_MODE
		mode &^= windows.ENABLE_INSERT_MODE
		mode &^= windows.ENABLE_MOUSE_INPUT
		mode &^= windows.ENABLE_EXTENDED_FLAGS

		// Enable these modes
		mode |= windows.ENABLE_WINDOW_INPUT
		mode |= windows.ENABLE_AUTO_POSITION

		inMode = mode
		windows.SetConsoleMode(in, inMode)
	} else {
		logger.Logger.Printf("failed to get console mode for stdin: %v\n", err)
	}
}
