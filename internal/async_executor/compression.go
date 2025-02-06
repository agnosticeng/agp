package async_executor

import (
	"compress/gzip"
	"fmt"
	"io"
)

func Compressor(codec ResultCompression, w io.Writer) (io.WriteCloser, error) {
	switch codec {
	case ResultCompressionNone:
		return &noopWriterCloser{w}, nil
	case ResultCompressionGZIP:
		return gzip.NewWriter(w), nil
	default:
		return nil, fmt.Errorf("unknown compression codec: %s", codec)
	}
}

type noopWriterCloser struct {
	io.Writer
}

func (w *noopWriterCloser) Close() error {
	return nil
}

func Decompressor(codec ResultCompression, r io.Reader) (io.ReadCloser, error) {
	switch codec {
	case ResultCompressionNone:
		return io.NopCloser(r), nil
	case ResultCompressionGZIP:
		return gzip.NewReader(r)
	default:
		return nil, fmt.Errorf("unknown compression codec: %s", codec)
	}
}
