//go:build windows

package packages

import (
	"math"
	"time"

	"golang.org/x/sys/windows"
)

// Windows process identity is obtained directly from Win32. In particular,
// this must not depend on an external shell being installed or launchable.
func windowsProcessStartTime(pid int) (string, processIdentityState) {
	started, ok, state := windowsProcessStart(pid)
	if !ok {
		return "", state
	}
	return "windows:" + started.UTC().Format(time.RFC3339Nano), processIdentityAvailable
}

func windowsProcessStartWallClock(pid int) (time.Time, bool) {
	started, ok, _ := windowsProcessStart(pid)
	return started, ok
}

func windowsProcessStart(pid int) (time.Time, bool, processIdentityState) {
	if pid <= 0 || uint64(pid) > math.MaxUint32 {
		return time.Time{}, false, processIdentityGone
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return time.Time{}, false, processIdentityGone
		}
		return time.Time{}, false, processIdentityUnavailable
	}
	defer windows.CloseHandle(h)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return time.Time{}, false, processIdentityUnavailable
	}
	// FILETIME is 100-nanosecond intervals since 1601-01-01 UTC.
	const windowsToUnixEpoch = 11644473600
	intervals := uint64(creation.HighDateTime)<<32 | uint64(creation.LowDateTime)
	if intervals == 0 {
		return time.Time{}, false, processIdentityUnavailable
	}
	seconds := int64(intervals/10_000_000) - windowsToUnixEpoch
	nanos := int64(intervals%10_000_000) * 100
	return time.Unix(seconds, nanos).UTC(), true, processIdentityAvailable
}
