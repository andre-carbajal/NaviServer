package addons

func curseForgeFingerprint(data []byte) uint32 {
	const m uint32 = 0x5bd1e995

	normalizedLen := 0
	for _, b := range data {
		if !isCurseWhitespace(b) {
			normalizedLen++
		}
	}

	hash := uint32(1) ^ uint32(normalizedLen)
	var carry uint32
	shift := 0

	for _, b := range data {
		if isCurseWhitespace(b) {
			continue
		}
		carry |= uint32(b) << shift
		shift += 8
		if shift == 32 {
			k := carry * m
			k ^= k >> 24
			k *= m

			hash *= m
			hash ^= k

			carry = 0
			shift = 0
		}
	}

	if shift > 0 {
		hash ^= carry
		hash *= m
	}

	hash ^= hash >> 13
	hash *= m
	hash ^= hash >> 15

	return hash
}

func isCurseWhitespace(value byte) bool {
	switch value {
	case 0x09, 0x0A, 0x0D, 0x20:
		return true
	default:
		return false
	}
}
