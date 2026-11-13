package main

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// sdCardWarning says, in plain words, when the database and thumbnails sit on the Pi's SD card: it is slow
// for thousands of small files and wears out. "" when they don't, or when it can't be told.
func sdCardWarning(dataDir string) string {
	var st unix.Stat_t
	if unix.Stat(dataDir, &st) != nil {
		return ""
	}
	mi, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	src := mountSource(string(mi), unix.Major(uint64(st.Dev)), unix.Minor(uint64(st.Dev)))
	if !strings.HasPrefix(src, "/dev/mmcblk") {
		return ""
	}
	return "The library data (" + dataDir + ") is on the SD card (" + src + "). Previews will be slow and the card wears out. " +
		"Move it to a USB SSD or hard drive: sudo dpkg-reconfigure picburrow (or start Picburrow with --data-dir)."
}

// mountSource finds the device behind major:minor in /proc/self/mountinfo
// ("36 25 179:2 / / rw,noatime shared:1 - ext4 /dev/mmcblk0p2 rw").
func mountSource(mountinfo string, major, minor uint32) string {
	want := strconv.FormatUint(uint64(major), 10) + ":" + strconv.FormatUint(uint64(minor), 10)
	for _, line := range strings.Split(mountinfo, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[2] != want {
			continue
		}
		for i, x := range f {
			if x == "-" && i+2 < len(f) {
				return f[i+2]
			}
		}
	}
	return ""
}
