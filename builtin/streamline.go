package builtin

import (
	"bufio"
	"bytes"
	"io"
	"github.com/WJQSERVER-STUDIO/go-utils/iox"
	"os"
	"regexp"
	"sync"
)

const streamedLineMemoryLimit = 64 * 1024

var bufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

type streamedLine struct {
	storage    *spillBuffer
	hadNewline bool
}

type spillBuffer struct {
	memory bytes.Buffer
	file   *os.File
	path   string
	size   int64
}

func readStreamedLine(reader *bufio.Reader) (*streamedLine, error) {
	line := &streamedLine{storage: &spillBuffer{}}

	for {
		chunk, err := reader.ReadSlice('\n')
		switch err {
		case nil:
			line.hadNewline = true
			payload := chunk[:len(chunk)-1]
			if writeErr := line.storage.Write(payload); writeErr != nil {
				line.Close()
				return nil, writeErr
			}
			return line, nil
		case bufio.ErrBufferFull:
			if writeErr := line.storage.Write(chunk); writeErr != nil {
				line.Close()
				return nil, writeErr
			}
		case io.EOF:
			if len(chunk) > 0 {
				if writeErr := line.storage.Write(chunk); writeErr != nil {
					line.Close()
					return nil, writeErr
				}
				return line, nil
			}
			if line.storage.Len() > 0 {
				return line, nil
			}
			line.Close()
			return nil, io.EOF
		default:
			line.Close()
			return nil, err
		}
	}
}

func (l *streamedLine) Close() error {
	if l == nil || l.storage == nil {
		return nil
	}
	return l.storage.Close()
}

func (l *streamedLine) Empty() bool {
	if l == nil || l.storage == nil {
		return true
	}
	return l.storage.Len() == 0
}

func (l *streamedLine) IsCRLFEmpty() bool {
	if l == nil || l.storage == nil {
		return true
	}
	if l.storage.Len() == 0 {
		return true
	}
	if l.hadNewline && l.storage.Len() == 1 {
		if l.storage.file == nil {
			data := l.storage.memory.Bytes()
			return len(data) == 1 && data[0] == '\r'
		}
		var buf [1]byte
		if _, err := l.storage.file.ReadAt(buf[:], 0); err == nil {
			return buf[0] == '\r'
		}
	}
	return false
}

func (l *streamedLine) MatchRegexp(pattern *regexp.Regexp) (bool, error) {
	if l.storage.file == nil {
		return pattern.Match(l.grepBytes()), nil
	}

	reader, err := l.grepReader()
	if err != nil {
		return false, err
	}
	return pattern.MatchReader(bufio.NewReader(reader)), nil
}

func (l *streamedLine) WriteTo(w io.Writer) error {
	return l.storage.WriteTo(w, l.storage.Len())
}

func (l *streamedLine) WriteForGrep(w io.Writer) error {
	if l.storage.file == nil {
		_, err := w.Write(l.grepBytes())
		return err
	}

	reader, err := l.grepReader()
	if err != nil {
		return err
	}
	_, err = iox.Copy(w, reader)
	return err
}

func (l *streamedLine) WriteProcessedTo(w io.Writer, showTabs, showNonprinting bool) error {
	if l.storage.file == nil {
		_, err := w.Write(processNonprinting(l.storage.memory.Bytes(), showTabs, showNonprinting))
		return err
	}

	reader, err := l.storage.Reader(l.storage.Len())
	if err != nil {
		return err
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	buf := *bufPtr

	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(processNonprinting(buf[:n], showTabs, showNonprinting)); writeErr != nil {
				return writeErr
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (l *streamedLine) grepBytes() []byte {
	data := l.storage.memory.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[:len(data)-1]
	}
	return data
}

func (l *streamedLine) grepReader() (io.Reader, error) {
	limit := l.storage.Len()
	if limit > 0 {
		var last byte
		if l.storage.file != nil {
			var buf [1]byte
			if _, err := l.storage.file.ReadAt(buf[:], limit-1); err == nil {
				last = buf[0]
			}
		} else {
			data := l.storage.memory.Bytes()
			last = data[len(data)-1]
		}
		if last == '\r' {
			limit--
		}
	}
	return l.storage.Reader(limit)
}

func (b *spillBuffer) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}

	if b.file == nil && b.size+int64(len(p)) <= streamedLineMemoryLimit {
		_, err := b.memory.Write(p)
		if err != nil {
			return err
		}
		b.size += int64(len(p))
		return nil
	}

	if b.file == nil {
		if err := b.spillToDisk(); err != nil {
			return err
		}
	}

	_, err := b.file.Write(p)
	if err != nil {
		return err
	}
	b.size += int64(len(p))
	return nil
}

func (b *spillBuffer) Len() int64 {
	return b.size
}

func (b *spillBuffer) Reader(limit int64) (io.Reader, error) {
	if b.file == nil {
		data := b.memory.Bytes()
		if limit < int64(len(data)) {
			data = data[:limit]
		}
		return bytes.NewReader(data), nil
	}

	return io.NewSectionReader(b.file, 0, limit), nil
}

func (b *spillBuffer) WriteTo(w io.Writer, limit int64) error {
	reader, err := b.Reader(limit)
	if err != nil {
		return err
	}
	_, err = iox.Copy(w, reader)
	return err
}

func (b *spillBuffer) Close() error {
	if b.file == nil {
		return nil
	}
	path := b.path
	err := b.file.Close()
	b.file = nil
	b.path = ""
	if removeErr := os.Remove(path); err == nil {
		err = removeErr
	}
	return err
}

func (b *spillBuffer) spillToDisk() error {
	file, err := os.CreateTemp("", "kamishell-line-*")
	if err != nil {
		return err
	}

	if _, err := file.Write(b.memory.Bytes()); err != nil {
		file.Close()
		os.Remove(file.Name())
		return err
	}

	b.file = file
	b.path = file.Name()
	b.size = int64(b.memory.Len())
	b.memory.Reset()
	return nil
}
