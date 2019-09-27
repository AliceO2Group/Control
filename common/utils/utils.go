package utils

import (
	"io"
	"os"
	"strings"
)

func IsDirEmpty(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close() //Read-only we don't care about the return value

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func EnsureTrailingSlash(path *string) {
	if !strings.HasSuffix(*path, "/") { //Add trailing '/'
		*path += "/"
	}
}