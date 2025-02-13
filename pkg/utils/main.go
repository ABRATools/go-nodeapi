package utils

func GetPtr[T any](t T) *T {
	return &t
}

func GetMapKeys(m map[uint16][]string) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
