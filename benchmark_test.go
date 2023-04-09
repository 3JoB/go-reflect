package reflect_test

import (
	"reflect"
	"testing"

	goreflect "github.com/3JoB/go-reflect"
)

func kindFromReflect(v any) reflect.Kind {
	return reflect.TypeOf(v).Kind()
}

func kindFromGoReflect(v any) goreflect.Kind {
	return goreflect.TypeOf(v).Kind()
}

func f(_ any) {}

func valueFromReflect(v any) {
	f(reflect.ValueOf(v).Elem())
}

func valueFromGoReflect(v any) {
	f(goreflect.ValueNoEscapeOf(v).Elem())
}

func Benchmark_TypeOf_Reflect(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var v struct {
			i int
		}
		kindFromReflect(&v)
	}
}

func Benchmark_TypeOf_GoReflect(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var v struct {
			i int
		}
		kindFromGoReflect(&v)
	}
}

func Benchmark_ValueOf_Reflect(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		valueFromReflect(&struct {
			I int
			F float64
		}{I: 10})
	}
}

func Benchmark_ValueOf_GoReflect(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		valueFromGoReflect(&struct {
			I int
			F float64
		}{I: 10})
	}
}
