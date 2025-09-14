package config

import (
	"io"
)

func Merge(data []io.Reader) (io.Reader, error) {
	return MergeJSON(data)
}
