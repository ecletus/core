package utils

type ReadonlyMapString interface {
	Get(key string) string
}
