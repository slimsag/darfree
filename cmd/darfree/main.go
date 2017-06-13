package main

import (
	"runtime"
	"runtime/debug"
	"time"

	// Comment out this line to see the difference. With this line here, you'll
	// observe ~17MB of memory usage. Without the comment, you'll notice ~529MB
	_ "github.com/slimsag/darfree"
)

func main() {
	for {
		// Allocate and use 512MB of memory.
		b := make([]byte, 512*1024*1024)
		for i := range b {
			b[i] = byte(i)
		}

		// GC and FreeOSMemory
		runtime.GC()
		debug.FreeOSMemory()

		time.Sleep(1 * time.Second)
	}
}
