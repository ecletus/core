package utils

type URLGenerator interface {
	GenStaticURL(path... string) string
	GenURL(path... string) string
	GenGlobalStaticURL(path... string) string
	GenGlobalURL(path... string) string
}

