package iokit

import "io"

func PrependReader(b []byte, r io.Reader) *PrependedReader {
	return &PrependedReader{b, r}
}

func PrependWriter(b []byte, w io.Writer) *PrependedWriter {
	return &PrependedWriter{b, w}
}

var _ io.Reader = &PrependedReader{}

type PrependedReader struct {
	prepend []byte
	R       io.Reader
}

func (p *PrependedReader) Read(b []byte) (int, error) {
	pn := len(p.prepend)
	if pn == 0 {
		return p.R.Read(b)
	}

	cn := copy(b, p.prepend)
	if cn == pn {
		if pr, ok := p.R.(*PrependedReader); ok {
			*p = *pr
			return cn, nil
		}
		p.prepend = nil
	} else {
		p.prepend = p.prepend[cn:]
	}

	if cn == len(b) {
		return cn, nil
	}
	n, err := p.R.Read(b[cn:])
	return cn + n, err
}

var _ io.WriteCloser = &PrependedWriter{}

type PrependedWriter struct {
	prepend []byte
	W       io.Writer
}

func (p *PrependedWriter) flush() error {
	if len(p.prepend) == 0 {
		return nil
	}
	fb := p.prepend
	p.prepend = nil
	switch n, err := p.W.Write(fb); {
	case err != nil:
		return err
	case n != len(fb):
		return io.ErrShortWrite
	default:
		return nil
	}
}

func (p *PrependedWriter) Write(b []byte) (int, error) {
	pn := len(p.prepend)
	if pn == 0 {
		return p.W.Write(b)
	}
	cp := cap(p.prepend) - pn
	bn := len(b)

	if bn < cp {
		p.prepend = append(p.prepend, b...)
		return bn, nil
	}

	p.prepend = append(p.prepend, b[:cp]...)
	fb := p.prepend
	p.prepend = nil
	switch n, err := p.W.Write(fb); {
	case err != nil:
		return n, err
	case n != len(fb):
		return n, io.ErrShortWrite
	}
	// TODO flattening

	if cp == len(b) {
		return cp, nil
	}
	n, err := p.W.Write(b[cp:])
	return cp + n, err
}

func (p *PrependedWriter) Flush() error {
	if err := p.flush(); err != nil {
		return err
	}
	if f, ok := p.W.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (p *PrependedWriter) Close() error {
	if err := p.flush(); err != nil {
		return err
	}
	if f, ok := p.W.(io.Closer); ok {
		return f.Close()
	}
	return nil
}
