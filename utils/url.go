package utils

type URLGenerator interface {
	JoinStaticURL(path ...string) string
	JoinPath(path ...string) string
}
