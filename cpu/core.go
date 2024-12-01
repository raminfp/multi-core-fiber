package cpu

import (
	"runtime"
	"time"
)

// GetCurrentCore Helper function to get current core (OS-level core detection)
func GetCurrentCore() int {
	return int(time.Now().UnixNano() % int64(runtime.NumCPU()))
}
