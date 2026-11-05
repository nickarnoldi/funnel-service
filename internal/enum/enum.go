package enum

// Index checks if the given value exists in the enumMap and returns the corresponding key and a boolean indicating existence.
func Index[T ~int](enumMap map[T]string, value string) (index int, found bool) {
	for k, v := range enumMap {
		if v == value {
			return int(k), true
		}
	}
	return -1, false
}
