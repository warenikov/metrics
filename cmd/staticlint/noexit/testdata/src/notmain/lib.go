package notmain

import "os"

// Exit is fine here — not in main package.
func Exit() {
	os.Exit(0)
}
