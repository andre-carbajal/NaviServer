package runner

import "naviserver/internal/jvm"

// GetJavaVersionForMC
// 1.20.5+ -> Java 21
// 1.18+   -> Java 17
// < 1.18  -> Java 8
func GetJavaVersionForMC(mcVersion string) int {
	return jvm.GetJavaVersionForMC(mcVersion)
}
