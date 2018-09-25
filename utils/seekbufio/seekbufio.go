package seekbufio

import (
	"bufio"
	"io"
	"os"
)

// ReadSeekerCloser is the interface that groups the basic Read, Seek, and Close methods.
type ReadSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// A seekable buffered reader, implementing ReadSeekerCloser
type SeekableBufferedReader struct {
	reader         ReadSeekerCloser
	bufferedReader *bufio.Reader
}

// Seek sets the offset for the next Read on seekable buffered reader to offset,
// interpreted according to whence. See the seek related constants in the io package
func (r SeekableBufferedReader) Seek(offset int64, whence int) (ret int64, err error) {
	if whence == io.SeekCurrent {
		offset -= int64(r.bufferedReader.Buffered())
	}
	if ret, err = r.reader.Seek(offset, whence); err != nil {
		return
	}
	r.bufferedReader.Reset(r.reader)
	return
}

// Read reads data into p. It returns the number of bytes read into p.
// The bytes are taken from at most one Read on the underlying Reader,
// hence n may be less than len(p). At EOF, the count will be zero and err
// will be io.EOF.
//
// See io.Reader and bufio.Reader
func (r SeekableBufferedReader) Read(p []byte) (n int, err error) {
	return r.bufferedReader.Read(p)
}

// Close closes the File, rendering it unusable for I/O
func (r SeekableBufferedReader) Close() error {
	return r.reader.Close()
}

// New returns a new SeekableBufferedReader from a ReadSeekerCloser,
// e.g. a os.File or similar type implementing ReadSeekerCloser
func New(file ReadSeekerCloser) SeekableBufferedReader {
	return SeekableBufferedReader{
		file,
		bufio.NewReader(file),
	}
}

// OpenSeekableBufferedReader a new SeekableBufferedReader by opening the named
// file in the filesystem.
func OpenSeekableBufferedReader(filename string) (SeekableBufferedReader, error) {
	return OpenSeekableBufferedReaderAt(filename, 0)
}

// OpenSeekableBufferedReader a new SeekableBufferedReader by opening the named
// file in the filesystem and seeking to at (relative to the start of the file)
func OpenSeekableBufferedReaderAt(filename string, at int64) (SeekableBufferedReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return SeekableBufferedReader{}, err
	}
	if _, err := file.Seek(at, io.SeekStart); err != nil {
		return SeekableBufferedReader{}, err
	}
	return New(file), nil
}
