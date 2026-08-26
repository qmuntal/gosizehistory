package platform

func CanonicalArch(arch string) string {
	if arch == "armv6" || arch == "armv6l" {
		return "arm"
	}
	return arch
}
