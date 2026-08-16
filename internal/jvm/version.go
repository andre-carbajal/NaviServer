package jvm

import (
	"strconv"
	"strings"
)

// GetJavaVersionForMC returns the minimum Java version required by a Minecraft version.
func GetJavaVersionForMC(mcVersion string) int {
	parts := strings.Split(mcVersion, ".")
	if len(parts) < 2 {
		return 21
	}

	first, _ := strconv.Atoi(parts[0])
	if first == 1 {
		minor, _ := strconv.Atoi(parts[1])

		if minor >= 20 {
			if len(parts) > 2 {
				patch, _ := strconv.Atoi(parts[2])
				if minor == 20 && patch >= 5 {
					return 21
				}
			}
			if minor >= 21 {
				return 21
			}
		}

		if minor >= 18 {
			return 17
		}

		return 8
	}

	if first >= 26 {
		return 25
	}

	return 21
}
