package utils

import "math"

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

func CalculateQuotaAndPeriod(numCPUs float64) (periodUS uint64, quotaUS int64) {
	const defaultPeriod = 100_000
	periodUS = defaultPeriod
	quotaUS = int64(math.Round(numCPUs * float64(periodUS)))
	return
}
