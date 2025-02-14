package reflect_test

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"unsafe"

	"github.com/3JoB/go-reflect"
)

var (
	typeToEncoderMap sync.Map
	bufpool          = sync.Pool{
		New: func() any {
			return &buffer{
				b: make([]byte, 0, 1024),
			}
		},
	}
)

type buffer struct {
	b []byte
}

type encoder func(*buffer, unsafe.Pointer) error

func Marshal(v any) ([]byte, error) {
	// Technique 1.
	// Get type information and pointer from interface{} value without allocation.
	typ, ptr := reflect.TypeAndPtrOf(v)
	typeID := reflect.TypeID(v)

	// Technique 2.
	// Reuse the buffer once allocated using sync.Pool
	buf := bufpool.Get().(*buffer)
	buf.b = buf.b[:0]
	defer bufpool.Put(buf)

	// Technique 3.
	// builds a optimized path by typeID and caches it
	if enc, ok := typeToEncoderMap.Load(typeID); ok {
		if err := enc.(encoder)(buf, ptr); err != nil {
			return nil, err
		}

		// allocate a new buffer required length only
		b := make([]byte, len(buf.b))
		copy(b, buf.b)
		return b, nil
	}

	// First time,
	// builds a optimized path by type and caches it with typeID.
	enc, err := compile(typ)
	if err != nil {
		return nil, err
	}
	typeToEncoderMap.Store(typeID, enc)
	if err := enc(buf, ptr); err != nil {
		return nil, err
	}

	// allocate a new buffer required length only
	b := make([]byte, len(buf.b))
	copy(b, buf.b)
	return b, nil
}

func compile(typ reflect.Type) (encoder, error) {
	switch typ.Kind() {
	case reflect.Struct:
		return compileStruct(typ)
	case reflect.Int:
		return compileInt(typ)
	}
	return nil, errors.New("unsupported type")
}

func compileStruct(typ reflect.Type) (encoder, error) {
	encoders := []encoder{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		enc, err := compile(field.Type)
		if err != nil {
			return nil, err
		}
		offset := field.Offset
		encoders = append(encoders, func(buf *buffer, p unsafe.Pointer) error {
			return enc(buf, unsafe.Pointer(uintptr(p)+offset))
		})
	}
	return func(buf *buffer, p unsafe.Pointer) error {
		buf.b = append(buf.b, '{')
		for i, enc := range encoders {
			if i != 0 {
				buf.b = append(buf.b, ' ')
			}
			if err := enc(buf, p); err != nil {
				return err
			}
		}
		buf.b = append(buf.b, '}')
		return nil
	}, nil
}

func compileInt(typ reflect.Type) (encoder, error) {
	return func(buf *buffer, p unsafe.Pointer) error {
		value := *(*int)(p)
		buf.b = strconv.AppendInt(buf.b, int64(value), 10)
		return nil
	}, nil
}

func Benchmark_Marshal(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		bytes, err := Marshal(struct{ I int }{I: 10})
		if err != nil {
			b.Fatal(err)
		}
		if string(bytes) != "{10}" {
			b.Fatalf("unexpected error: %s", string(bytes))
		}
		bytes2, err := Marshal(struct{ I, J int }{I: 10, J: 20})
		if err != nil {
			b.Fatal(err)
		}
		if string(bytes2) != "{10 20}" {
			b.Fatalf("unexpected error: %s", string(bytes2))
		}
	}
}
