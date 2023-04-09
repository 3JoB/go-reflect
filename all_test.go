// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect_test

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	. "github.com/3JoB/go-reflect"
)

const (
	PtrSize = 8
)

var sink any

func TestBool(t *testing.T) {
	v := ValueOf(true)
	if !v.Bool() {
		t.Fatal("ValueOf(true).Bool() = false")
	}
}

type integer int

type T struct {
	a int
	b float64
	c string
	d *int
}

type pair struct {
	i any
	s string
}

func assert(t *testing.T, s, want string) {
	if s != want {
		t.Errorf("have %#q want %#q", s, want)
	}
}

var typeTests = []pair{
	{i: struct{ x int }{}, s: "int"},
	{i: struct{ x int8 }{}, s: "int8"},
	{i: struct{ x int16 }{}, s: "int16"},
	{i: struct{ x int32 }{}, s: "int32"},
	{i: struct{ x int64 }{}, s: "int64"},
	{i: struct{ x uint }{}, s: "uint"},
	{i: struct{ x uint8 }{}, s: "uint8"},
	{i: struct{ x uint16 }{}, s: "uint16"},
	{i: struct{ x uint32 }{}, s: "uint32"},
	{i: struct{ x uint64 }{}, s: "uint64"},
	{i: struct{ x float32 }{}, s: "float32"},
	{i: struct{ x float64 }{}, s: "float64"},
	{i: struct{ x int8 }{}, s: "int8"},
	{i: struct{ x (**int8) }{}, s: "**int8"},
	{i: struct{ x (**integer) }{}, s: "**reflect_test.integer"},
	{i: struct{ x ([32]int32) }{}, s: "[32]int32"},
	{i: struct{ x ([]int8) }{}, s: "[]int8"},
	{i: struct{ x (map[string]int32) }{}, s: "map[string]int32"},
	{i: struct{ x (chan<- string) }{}, s: "chan<- string"},
	{i: struct {
		x struct {
			c chan *int32
			d float32
		}
	}{},
		s: "struct { c chan *int32; d float32 }",
	},
	{i: struct{ x (func(a int8, b int32)) }{}, s: "func(int8, int32)"},
	{i: struct {
		x struct {
			c func(chan *integer, *int8)
		}
	}{},
		s: "struct { c func(chan *reflect_test.integer, *int8) }",
	},
	{i: struct {
		x struct {
			a int8
			b int32
		}
	}{},
		s: "struct { a int8; b int32 }",
	},
	{i: struct {
		x struct {
			a int8
			b int8
			c int32
		}
	}{},
		s: "struct { a int8; b int8; c int32 }",
	},
	{i: struct {
		x struct {
			a int8
			b int8
			c int8
			d int32
		}
	}{},
		s: "struct { a int8; b int8; c int8; d int32 }",
	},
	{i: struct {
		x struct {
			a int8
			b int8
			c int8
			d int8
			e int32
		}
	}{},
		s: "struct { a int8; b int8; c int8; d int8; e int32 }",
	},
	{i: struct {
		x struct {
			a int8
			b int8
			c int8
			d int8
			e int8
			f int32
		}
	}{},
		s: "struct { a int8; b int8; c int8; d int8; e int8; f int32 }",
	},
	{i: struct {
		x struct {
			a int8 `reflect:"hi there"`
		}
	}{},
		s: `struct { a int8 "reflect:\"hi there\"" }`,
	},
	{i: struct {
		x struct {
			a int8 `reflect:"hi \x00there\t\n\"\\"`
		}
	}{},
		s: `struct { a int8 "reflect:\"hi \\x00there\\t\\n\\\"\\\\\"" }`,
	},
	{i: struct {
		x struct {
			f func(args ...int)
		}
	}{},
		s: "struct { f func(...int) }",
	},
	{i: struct {
		x (interface {
			a(func(func(int) int) func(func(int)) int)
			b()
		})
	}{},
		s: "interface { reflect_test.a(func(func(int) int) func(func(int)) int); reflect_test.b() }",
	},
	{i: struct {
		x struct {
			int32
			int64
		}
	}{},
		s: "struct { int32; int64 }",
	},
}

var valueTests = []pair{
	{i: new(int), s: "132"},
	{i: new(int8), s: "8"},
	{i: new(int16), s: "16"},
	{i: new(int32), s: "32"},
	{i: new(int64), s: "64"},
	{i: new(uint), s: "132"},
	{i: new(uint8), s: "8"},
	{i: new(uint16), s: "16"},
	{i: new(uint32), s: "32"},
	{i: new(uint64), s: "64"},
	{i: new(float32), s: "256.25"},
	{i: new(float64), s: "512.125"},
	{i: new(complex64), s: "532.125+10i"},
	{i: new(complex128), s: "564.25+1i"},
	{i: new(string), s: "stringy cheese"},
	{i: new(bool), s: "true"},
	{i: new(*int8), s: "*int8(0)"},
	{i: new(**int8), s: "**int8(0)"},
	{i: new([5]int32), s: "[5]int32{0, 0, 0, 0, 0}"},
	{i: new(**integer), s: "**reflect_test.integer(0)"},
	{i: new(map[string]int32), s: "map[string]int32{<can't iterate on maps>}"},
	{i: new(chan<- string), s: "chan<- string"},
	{i: new(func(a int8, b int32)), s: "func(int8, int32)(0)"},
	{i: new(struct {
		c chan *int32
		d float32
	}),
		s: "struct { c chan *int32; d float32 }{chan *int32, 0}",
	},
	{i: new(struct{ c func(chan *integer, *int8) }),
		s: "struct { c func(chan *reflect_test.integer, *int8) }{func(chan *reflect_test.integer, *int8)(0)}",
	},
	{i: new(struct {
		a int8
		b int32
	}),
		s: "struct { a int8; b int32 }{0, 0}",
	},
	{i: new(struct {
		a int8
		b int8
		c int32
	}),
		s: "struct { a int8; b int8; c int32 }{0, 0, 0}",
	},
}

func testType(t *testing.T, i int, typ Type, want string) {
	s := typ.String()
	if s != want {
		t.Errorf("#%d: have %#q, want %#q", i, s, want)
	}
}

func TestTypes(t *testing.T) {
	for i, tt := range typeTests {
		testType(t, i, ValueOf(tt.i).Field(0).Type(), tt.s)
	}
}

func TestSet(t *testing.T) {
	for i, tt := range valueTests {
		v := ValueOf(tt.i)
		v = v.Elem()
		switch v.Kind() {
		case Int:
			v.SetInt(132)
		case Int8:
			v.SetInt(8)
		case Int16:
			v.SetInt(16)
		case Int32:
			v.SetInt(32)
		case Int64:
			v.SetInt(64)
		case Uint:
			v.SetUint(132)
		case Uint8:
			v.SetUint(8)
		case Uint16:
			v.SetUint(16)
		case Uint32:
			v.SetUint(32)
		case Uint64:
			v.SetUint(64)
		case Float32:
			v.SetFloat(256.25)
		case Float64:
			v.SetFloat(512.125)
		case Complex64:
			v.SetComplex(532.125 + 10i)
		case Complex128:
			v.SetComplex(564.25 + 1i)
		case String:
			v.SetString("stringy cheese")
		case Bool:
			v.SetBool(true)
		}
		s := valueToString(v)
		if s != tt.s {
			t.Errorf("#%d: have %#q, want %#q", i, s, tt.s)
		}
	}
}

func TestSetValue(t *testing.T) {
	for i, tt := range valueTests {
		v := ValueOf(tt.i).Elem()
		switch v.Kind() {
		case Int:
			v.Set(ValueOf(int(132)))
		case Int8:
			v.Set(ValueOf(int8(8)))
		case Int16:
			v.Set(ValueOf(int16(16)))
		case Int32:
			v.Set(ValueOf(int32(32)))
		case Int64:
			v.Set(ValueOf(int64(64)))
		case Uint:
			v.Set(ValueOf(uint(132)))
		case Uint8:
			v.Set(ValueOf(uint8(8)))
		case Uint16:
			v.Set(ValueOf(uint16(16)))
		case Uint32:
			v.Set(ValueOf(uint32(32)))
		case Uint64:
			v.Set(ValueOf(uint64(64)))
		case Float32:
			v.Set(ValueOf(float32(256.25)))
		case Float64:
			v.Set(ValueOf(512.125))
		case Complex64:
			v.Set(ValueOf(complex64(532.125 + 10i)))
		case Complex128:
			v.Set(ValueOf(complex128(564.25 + 1i)))
		case String:
			v.Set(ValueOf("stringy cheese"))
		case Bool:
			v.Set(ValueOf(true))
		}
		s := valueToString(v)
		if s != tt.s {
			t.Errorf("#%d: have %#q, want %#q", i, s, tt.s)
		}
	}
}

func TestCanSetField(t *testing.T) {
	type embed struct{ x, X int }
	type Embed struct{ x, X int }
	type S1 struct {
		embed
		x, X int
	}
	type S2 struct {
		*embed
		x, X int
	}
	type S3 struct {
		Embed
		x, X int
	}
	type S4 struct {
		*Embed
		x, X int
	}

	type testCase struct {
		index  []int
		canSet bool
	}
	tests := []struct {
		val   Value
		cases []testCase
	}{{
		val: ValueOf(&S1{}),
		cases: []testCase{
			{index: []int{0}, canSet: false},
			{index: []int{0, 0}, canSet: false},
			{index: []int{0, 1}, canSet: true},
			{index: []int{1}, canSet: false},
			{index: []int{2}, canSet: true},
		},
	}, {
		val: ValueOf(&S2{embed: &embed{}}),
		cases: []testCase{
			{index: []int{0}, canSet: false},
			{index: []int{0, 0}, canSet: false},
			{index: []int{0, 1}, canSet: true},
			{index: []int{1}, canSet: false},
			{index: []int{2}, canSet: true},
		},
	}, {
		val: ValueOf(&S3{}),
		cases: []testCase{
			{index: []int{0}, canSet: true},
			{index: []int{0, 0}, canSet: false},
			{index: []int{0, 1}, canSet: true},
			{index: []int{1}, canSet: false},
			{index: []int{2}, canSet: true},
		},
	}, {
		val: ValueOf(&S4{Embed: &Embed{}}),
		cases: []testCase{
			{index: []int{0}, canSet: true},
			{index: []int{0, 0}, canSet: false},
			{index: []int{0, 1}, canSet: true},
			{index: []int{1}, canSet: false},
			{index: []int{2}, canSet: true},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.val.Type().Name(), func(t *testing.T) {
			for _, tc := range tt.cases {
				f := tt.val
				for _, i := range tc.index {
					if f.Kind() == Ptr {
						f = f.Elem()
					}
					f = f.Field(i)
				}
				if got := f.CanSet(); got != tc.canSet {
					t.Errorf("CanSet() = %v, want %v", got, tc.canSet)
				}
			}
		})
	}
}

var _i = 7

var valueToStringTests = []pair{
	{i: 123, s: "123"},
	{i: 123.5, s: "123.5"},
	{i: byte(123), s: "123"},
	{i: "abc", s: "abc"},
	{i: T{a: 123, b: 456.75, c: "hello", d: &_i}, s: "reflect_test.T{123, 456.75, hello, *int(&7)}"},
	{i: new(chan *T), s: "*chan *reflect_test.T(&chan *reflect_test.T)"},
	{i: [10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, s: "[10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}"},
	{i: &[10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, s: "*[10]int(&[10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})"},
	{i: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, s: "[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}"},
	{i: &[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, s: "*[]int(&[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})"},
}

func TestValueToString(t *testing.T) {
	for i, test := range valueToStringTests {
		s := valueToString(ValueOf(test.i))
		if s != test.s {
			t.Errorf("#%d: have %#q, want %#q", i, s, test.s)
		}
	}
}

func TestArrayElemSet(t *testing.T) {
	v := ValueOf(&[10]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}).Elem()
	v.Index(4).SetInt(123)
	s := valueToString(v)
	const want = "[10]int{1, 2, 3, 4, 123, 6, 7, 8, 9, 10}"
	if s != want {
		t.Errorf("[10]int: have %#q want %#q", s, want)
	}

	v = ValueOf([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	v.Index(4).SetInt(123)
	s = valueToString(v)
	const want1 = "[]int{1, 2, 3, 4, 123, 6, 7, 8, 9, 10}"
	if s != want1 {
		t.Errorf("[]int: have %#q want %#q", s, want1)
	}
}

func TestPtrPointTo(t *testing.T) {
	var ip *int32
	var i int32 = 1234
	vip := ValueOf(&ip)
	vi := ValueOf(&i).Elem()
	vip.Elem().Set(vi.Addr())
	if *ip != 1234 {
		t.Errorf("got %d, want 1234", *ip)
	}

	ip = nil
	vp := ValueOf(&ip).Elem()
	vp.Set(Zero(vp.Type()))
	if ip != nil {
		t.Errorf("got non-nil (%p), want nil", ip)
	}
}

func TestPtrSetNil(t *testing.T) {
	var i int32 = 1234
	ip := &i
	vip := ValueOf(&ip)
	vip.Elem().Set(Zero(vip.Elem().Type()))
	if ip != nil {
		t.Errorf("got non-nil (%d), want nil", *ip)
	}
}

func TestMapSetNil(t *testing.T) {
	m := make(map[string]int)
	vm := ValueOf(&m)
	vm.Elem().Set(Zero(vm.Elem().Type()))
	if m != nil {
		t.Errorf("got non-nil (%p), want nil", m)
	}
}

func TestAll(t *testing.T) {
	testType(t, 1, TypeOf((int8)(0)), "int8")
	testType(t, 2, TypeOf((*int8)(nil)).Elem(), "int8")

	typ := TypeOf((*struct {
		c chan *int32
		d float32
	})(nil))
	testType(t, 3, typ, "*struct { c chan *int32; d float32 }")
	etyp := typ.Elem()
	testType(t, 4, etyp, "struct { c chan *int32; d float32 }")
	styp := etyp
	f := styp.Field(0)
	testType(t, 5, f.Type, "chan *int32")

	f, present := styp.FieldByName("d")
	if !present {
		t.Errorf("FieldByName says present field is absent")
	}
	testType(t, 6, f.Type, "float32")

	f, present = styp.FieldByName("absent")
	if present {
		t.Errorf("FieldByName says absent field is present")
	}

	typ = TypeOf([32]int32{})
	testType(t, 7, typ, "[32]int32")
	testType(t, 8, typ.Elem(), "int32")

	typ = TypeOf((map[string]*int32)(nil))
	testType(t, 9, typ, "map[string]*int32")
	mtyp := typ
	testType(t, 10, mtyp.Key(), "string")
	testType(t, 11, mtyp.Elem(), "*int32")

	typ = TypeOf((chan<- string)(nil))
	testType(t, 12, typ, "chan<- string")
	testType(t, 13, typ.Elem(), "string")

	// make sure tag strings are not part of element type
	typ = TypeOf(struct {
		d []uint32 `reflect:"TAG"`
	}{}).Field(0).Type
	testType(t, 14, typ, "[]uint32")
}

func TestInterfaceGet(t *testing.T) {
	var inter struct {
		E any
	}
	inter.E = 123.456
	v1 := ValueOf(&inter)
	v2 := v1.Elem().Field(0)
	assert(t, v2.Type().String(), "interface {}")
	i2 := v2.Interface()
	v3 := ValueOf(i2)
	assert(t, v3.Type().String(), "float64")
}

func TestInterfaceValue(t *testing.T) {
	var inter struct {
		E any
	}
	inter.E = 123.456
	v1 := ValueOf(&inter)
	v2 := v1.Elem().Field(0)
	assert(t, v2.Type().String(), "interface {}")
	v3 := v2.Elem()
	assert(t, v3.Type().String(), "float64")

	i3 := v2.Interface()
	if _, ok := i3.(float64); !ok {
		t.Error("v2.Interface() did not return float64, got ", TypeOf(i3))
	}
}

func TestFunctionValue(t *testing.T) {
	var x any = func() {}
	v := ValueOf(x)
	if fmt.Sprint(v.Interface()) != fmt.Sprint(x) {
		t.Fatalf("TestFunction returned wrong pointer")
	}
	assert(t, v.Type().String(), "func()")
}

var appendTests = []struct {
	orig, extra []int
}{
	{orig: make([]int, 2, 4), extra: []int{22}},
	{orig: make([]int, 2, 4), extra: []int{22, 33, 44}},
}

func sameInts(x, y []int) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xx := range x {
		if xx != y[i] {
			return false
		}
	}
	return true
}

func TestAppend(t *testing.T) {
	for i, test := range appendTests {
		origLen, extraLen := len(test.orig), len(test.extra)
		want := append(test.orig, test.extra...)
		// Convert extra from []int to []Value.
		e0 := make([]Value, len(test.extra))
		for j, e := range test.extra {
			e0[j] = ValueOf(e)
		}
		// Convert extra from []int to *SliceValue.
		e1 := ValueOf(test.extra)
		// Test Append.
		a0 := ValueOf(test.orig)
		have0 := Append(a0, e0...).Interface().([]int)
		if !sameInts(have0, want) {
			t.Errorf("Append #%d: have %v, want %v (%p %p)", i, have0, want, test.orig, have0)
		}
		// Check that the orig and extra slices were not modified.
		if len(test.orig) != origLen {
			t.Errorf("Append #%d origLen: have %v, want %v", i, len(test.orig), origLen)
		}
		if len(test.extra) != extraLen {
			t.Errorf("Append #%d extraLen: have %v, want %v", i, len(test.extra), extraLen)
		}
		// Test AppendSlice.
		a1 := ValueOf(test.orig)
		have1 := AppendSlice(a1, e1).Interface().([]int)
		if !sameInts(have1, want) {
			t.Errorf("AppendSlice #%d: have %v, want %v", i, have1, want)
		}
		// Check that the orig and extra slices were not modified.
		if len(test.orig) != origLen {
			t.Errorf("AppendSlice #%d origLen: have %v, want %v", i, len(test.orig), origLen)
		}
		if len(test.extra) != extraLen {
			t.Errorf("AppendSlice #%d extraLen: have %v, want %v", i, len(test.extra), extraLen)
		}
	}
}

func TestCopy(t *testing.T) {
	a := []int{1, 2, 3, 4, 10, 9, 8, 7}
	b := []int{11, 22, 33, 44, 1010, 99, 88, 77, 66, 55, 44}
	c := []int{11, 22, 33, 44, 1010, 99, 88, 77, 66, 55, 44}
	for i := 0; i < len(b); i++ {
		if b[i] != c[i] {
			t.Fatalf("b != c before test")
		}
	}
	a1 := a
	b1 := b
	aa := ValueOf(&a1).Elem()
	ab := ValueOf(&b1).Elem()
	for tocopy := 1; tocopy <= 7; tocopy++ {
		aa.SetLen(tocopy)
		Copy(ab, aa)
		aa.SetLen(8)
		for i := 0; i < tocopy; i++ {
			if a[i] != b[i] {
				t.Errorf("(i) tocopy=%d a[%d]=%d, b[%d]=%d",
					tocopy, i, a[i], i, b[i])
			}
		}
		for i := tocopy; i < len(b); i++ {
			if b[i] != c[i] {
				if i < len(a) {
					t.Errorf("(ii) tocopy=%d a[%d]=%d, b[%d]=%d, c[%d]=%d",
						tocopy, i, a[i], i, b[i], i, c[i])
				} else {
					t.Errorf("(iii) tocopy=%d b[%d]=%d, c[%d]=%d",
						tocopy, i, b[i], i, c[i])
				}
			} else {
				t.Logf("tocopy=%d elem %d is okay\n", tocopy, i)
			}
		}
	}
}

func TestCopyString(t *testing.T) {
	t.Run("Slice", func(t *testing.T) {
		s := bytes.Repeat([]byte{'_'}, 8)
		val := ValueOf(s)

		n := Copy(val, ValueOf(""))
		if expecting := []byte("________"); n != 0 || !bytes.Equal(s, expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 0, s = %s", n, s, expecting)
		}

		n = Copy(val, ValueOf("hello"))
		if expecting := []byte("hello___"); n != 5 || !bytes.Equal(s, expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 5, s = %s", n, s, expecting)
		}

		n = Copy(val, ValueOf("helloworld"))
		if expecting := []byte("hellowor"); n != 8 || !bytes.Equal(s, expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 8, s = %s", n, s, expecting)
		}
	})
	t.Run("Array", func(t *testing.T) {
		s := [...]byte{'_', '_', '_', '_', '_', '_', '_', '_'}
		val := ValueOf(&s).Elem()

		n := Copy(val, ValueOf(""))
		if expecting := []byte("________"); n != 0 || !bytes.Equal(s[:], expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 0, s = %s", n, s[:], expecting)
		}

		n = Copy(val, ValueOf("hello"))
		if expecting := []byte("hello___"); n != 5 || !bytes.Equal(s[:], expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 5, s = %s", n, s[:], expecting)
		}

		n = Copy(val, ValueOf("helloworld"))
		if expecting := []byte("hellowor"); n != 8 || !bytes.Equal(s[:], expecting) {
			t.Errorf("got n = %d, s = %s, expecting n = 8, s = %s", n, s[:], expecting)
		}
	})
}

func TestCopyArray(t *testing.T) {
	a := [8]int{1, 2, 3, 4, 10, 9, 8, 7}
	b := [11]int{11, 22, 33, 44, 1010, 99, 88, 77, 66, 55, 44}
	c := b
	aa := ValueOf(&a).Elem()
	ab := ValueOf(&b).Elem()
	Copy(ab, aa)
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			t.Errorf("(i) a[%d]=%d, b[%d]=%d", i, a[i], i, b[i])
		}
	}
	for i := len(a); i < len(b); i++ {
		if b[i] != c[i] {
			t.Errorf("(ii) b[%d]=%d, c[%d]=%d", i, b[i], i, c[i])
		} else {
			t.Logf("elem %d is okay\n", i)
		}
	}
}

func TestBigUnnamedStruct(t *testing.T) {
	b := struct{ a, b, c, d int64 }{a: 1, b: 2, c: 3, d: 4}
	v := ValueOf(b)
	b1 := v.Interface().(struct {
		a, b, c, d int64
	})
	if b1.a != b.a || b1.b != b.b || b1.c != b.c || b1.d != b.d {
		t.Errorf("ValueOf(%v).Interface().(*Big) = %v", b, b1)
	}
}

type big struct {
	a, b, c, d, e int64
}

func TestBigStruct(t *testing.T) {
	b := big{a: 1, b: 2, c: 3, d: 4, e: 5}
	v := ValueOf(b)
	b1 := v.Interface().(big)
	if b1.a != b.a || b1.b != b.b || b1.c != b.c || b1.d != b.d || b1.e != b.e {
		t.Errorf("ValueOf(%v).Interface().(big) = %v", b, b1)
	}
}

type Basic struct {
	x int
	y float32
}

type NotBasic Basic

type DeepEqualTest struct {
	a, b any
	eq   bool
}

// Simple functions for DeepEqual tests.
var (
	fn1 func()             // nil.
	fn2 func()             // nil.
	fn3 = func() { fn1() } // Not nil.
)

type self struct{}

type Loop *Loop

type Loopy any

var loop1, loop2 Loop
var loopy1, loopy2 Loopy

func init() {
	loop1 = &loop2
	loop2 = &loop1

	loopy1 = &loopy2
	loopy2 = &loopy1
}

var deepEqualTests = []DeepEqualTest{
	// Equalities
	{a: nil, b: nil, eq: true},
	{a: 1, b: 1, eq: true},
	{a: int32(1), b: int32(1), eq: true},
	{a: 0.5, b: 0.5, eq: true},
	{a: float32(0.5), b: float32(0.5), eq: true},
	{a: "hello", b: "hello", eq: true},
	{a: make([]int, 10), b: make([]int, 10), eq: true},
	{a: &[3]int{1, 2, 3}, b: &[3]int{1, 2, 3}, eq: true},
	{a: Basic{x: 1, y: 0.5}, b: Basic{x: 1, y: 0.5}, eq: true},
	{a: error(nil), b: error(nil), eq: true},
	{a: map[int]string{1: "one", 2: "two"}, b: map[int]string{2: "two", 1: "one"}, eq: true},
	{a: fn1, b: fn2, eq: true},

	// Inequalities
	{a: 1, b: 2, eq: false},
	{a: int32(1), b: int32(2), eq: false},
	{a: 0.5, b: 0.6, eq: false},
	{a: float32(0.5), b: float32(0.6), eq: false},
	{a: "hello", b: "hey", eq: false},
	{a: make([]int, 10), b: make([]int, 11), eq: false},
	{a: &[3]int{1, 2, 3}, b: &[3]int{1, 2, 4}, eq: false},
	{a: Basic{x: 1, y: 0.5}, b: Basic{x: 1, y: 0.6}, eq: false},
	{a: Basic{x: 1, y: 0}, b: Basic{x: 2, y: 0}, eq: false},
	{a: map[int]string{1: "one", 3: "two"}, b: map[int]string{2: "two", 1: "one"}, eq: false},
	{a: map[int]string{1: "one", 2: "txo"}, b: map[int]string{2: "two", 1: "one"}, eq: false},
	{a: map[int]string{1: "one"}, b: map[int]string{2: "two", 1: "one"}, eq: false},
	{a: map[int]string{2: "two", 1: "one"}, b: map[int]string{1: "one"}, eq: false},
	{a: nil, b: 1, eq: false},
	{a: 1, b: nil, eq: false},
	{a: fn1, b: fn3, eq: false},
	{a: fn3, b: fn3, eq: false},
	{a: [][]int{{1}}, b: [][]int{{2}}, eq: false},
	{a: math.NaN(), b: math.NaN(), eq: false},
	{a: &[1]float64{math.NaN()}, b: &[1]float64{math.NaN()}, eq: false},
	{a: &[1]float64{math.NaN()}, b: self{}, eq: true},
	{a: []float64{math.NaN()}, b: []float64{math.NaN()}, eq: false},
	{a: []float64{math.NaN()}, b: self{}, eq: true},
	{a: map[float64]float64{math.NaN(): 1}, b: map[float64]float64{1: 2}, eq: false},
	{a: map[float64]float64{math.NaN(): 1}, b: self{}, eq: true},

	// Nil vs empty: not the same.
	{a: []int{}, b: []int(nil), eq: false},
	{a: []int{}, b: []int{}, eq: true},
	{a: []int(nil), b: []int(nil), eq: true},
	{a: map[int]int{}, b: map[int]int(nil), eq: false},
	{a: map[int]int{}, b: map[int]int{}, eq: true},
	{a: map[int]int(nil), b: map[int]int(nil), eq: true},

	// Mismatched types
	{a: 1, b: 1.0, eq: false},
	{a: int32(1), b: int64(1), eq: false},
	{a: 0.5, b: "hello", eq: false},
	{a: []int{1, 2, 3}, b: [3]int{1, 2, 3}, eq: false},
	{a: &[3]any{1, 2, 4}, b: &[3]any{1, 2, "s"}, eq: false},
	{a: Basic{x: 1, y: 0.5}, b: NotBasic{x: 1, y: 0.5}, eq: false},
	{a: map[uint]string{1: "one", 2: "two"}, b: map[int]string{2: "two", 1: "one"}, eq: false},

	// Possible loops.
	{a: &loop1, b: &loop1, eq: true},
	{a: &loop1, b: &loop2, eq: true},
	{a: &loopy1, b: &loopy1, eq: true},
	{a: &loopy1, b: &loopy2, eq: true},
}

func TestDeepEqual(t *testing.T) {
	for _, test := range deepEqualTests {
		if test.b == (self{}) {
			test.b = test.a
		}
		if r := DeepEqual(test.a, test.b); r != test.eq {
			t.Errorf("DeepEqual(%v, %v) = %v, want %v", test.a, test.b, r, test.eq)
		}
	}
}

func TestTypeOf(t *testing.T) {
	// Special case for nil
	if typ := TypeOf(nil); typ != nil {
		t.Errorf("expected nil type for nil value; got %v", typ)
	}
	for _, test := range deepEqualTests {
		v := ValueOf(test.a)
		if !v.IsValid() {
			continue
		}
		typ := TypeOf(test.a)
		if typ != v.Type() {
			t.Errorf("TypeOf(%v) = %v, but ValueOf(%v).Type() = %v", test.a, typ, test.a, v.Type())
		}
	}
}

type Recursive struct {
	x int
	r *Recursive
}

func TestDeepEqualRecursiveStruct(t *testing.T) {
	a, b := new(Recursive), new(Recursive)
	*a = Recursive{x: 12, r: a}
	*b = Recursive{x: 12, r: b}
	if !DeepEqual(a, b) {
		t.Error("DeepEqual(recursive same) = false, want true")
	}
}

type _Complex struct {
	a int
	b [3]*_Complex
	c *string
	d map[float64]float64
}

func TestDeepEqualComplexStruct(t *testing.T) {
	m := make(map[float64]float64)
	stra, strb := "hello", "hello"
	a, b := new(_Complex), new(_Complex)
	*a = _Complex{a: 5, b: [3]*_Complex{a, b, a}, c: &stra, d: m}
	*b = _Complex{a: 5, b: [3]*_Complex{b, a, a}, c: &strb, d: m}
	if !DeepEqual(a, b) {
		t.Error("DeepEqual(complex same) = false, want true")
	}
}

func TestDeepEqualComplexStructInequality(t *testing.T) {
	m := make(map[float64]float64)
	stra, strb := "hello", "helloo" // Difference is here
	a, b := new(_Complex), new(_Complex)
	*a = _Complex{a: 5, b: [3]*_Complex{a, b, a}, c: &stra, d: m}
	*b = _Complex{a: 5, b: [3]*_Complex{b, a, a}, c: &strb, d: m}
	if DeepEqual(a, b) {
		t.Error("DeepEqual(complex different) = true, want false")
	}
}

type UnexpT struct {
	m map[int]int
}

func TestDeepEqualUnexportedMap(t *testing.T) {
	// Check that DeepEqual can look at unexported fields.
	x1 := UnexpT{m: map[int]int{1: 2}}
	x2 := UnexpT{m: map[int]int{1: 2}}
	if !DeepEqual(&x1, &x2) {
		t.Error("DeepEqual(x1, x2) = false, want true")
	}

	y1 := UnexpT{m: map[int]int{2: 3}}
	if DeepEqual(&x1, &y1) {
		t.Error("DeepEqual(x1, y1) = true, want false")
	}
}

func check2ndField(x any, offs uintptr, t *testing.T) {
	s := ValueOf(x)
	f := s.Type().Field(1)
	if f.Offset != offs {
		t.Error("mismatched offsets in structure alignment:", f.Offset, offs)
	}
}

// Check that structure alignment & offsets viewed through reflect agree with those
// from the compiler itself.
func TestAlignment(t *testing.T) {
	type T1inner struct {
		a int
	}
	type T1 struct {
		T1inner
		f int
	}
	type T2inner struct {
		a, b int
	}
	type T2 struct {
		T2inner
		f int
	}

	x := T1{T1inner: T1inner{a: 2}, f: 17}
	check2ndField(x, uintptr(unsafe.Pointer(&x.f))-uintptr(unsafe.Pointer(&x)), t)

	x1 := T2{T2inner: T2inner{a: 2, b: 3}, f: 17}
	check2ndField(x1, uintptr(unsafe.Pointer(&x1.f))-uintptr(unsafe.Pointer(&x1)), t)
}

func Nil(a any, t *testing.T) {
	n := ValueOf(a).Field(0)
	if !n.IsNil() {
		t.Errorf("%v should be nil", a)
	}
}

func NotNil(a any, t *testing.T) {
	n := ValueOf(a).Field(0)
	if n.IsNil() {
		t.Errorf("value of type %v should not be nil", ValueOf(a).Type().String())
	}
}

func TestIsNil(t *testing.T) {
	// These implement IsNil.
	// Wrap in extra struct to hide interface type.
	doNil := []any{
		struct{ x *int }{},
		struct{ x any }{},
		struct{ x map[string]int }{},
		struct{ x func() bool }{},
		struct{ x chan int }{},
		struct{ x []string }{},
		struct{ x unsafe.Pointer }{},
	}
	for _, ts := range doNil {
		ty := TypeOf(ts).Field(0).Type
		v := Zero(ty)
		v.IsNil() // panics if not okay to call
	}

	// Check the implementations
	var pi struct {
		x *int
	}
	Nil(pi, t)
	pi.x = new(int)
	NotNil(pi, t)

	var si struct {
		x []int
	}
	Nil(si, t)
	si.x = make([]int, 10)
	NotNil(si, t)

	var ci struct {
		x chan int
	}
	Nil(ci, t)
	ci.x = make(chan int)
	NotNil(ci, t)

	var mi struct {
		x map[int]int
	}
	Nil(mi, t)
	mi.x = make(map[int]int)
	NotNil(mi, t)

	var ii struct {
		x any
	}
	Nil(ii, t)
	ii.x = 2
	NotNil(ii, t)

	var fi struct {
		x func(t *testing.T)
	}
	Nil(fi, t)
	fi.x = TestIsNil
	NotNil(fi, t)
}

func TestInterfaceExtraction(t *testing.T) {
	var s struct {
		W io.Writer
	}

	s.W = os.Stdout
	v := Indirect(ValueOf(&s)).Field(0).Interface()
	if v != s.W.(any) {
		t.Error("Interface() on interface: ", v, s.W)
	}
}

func TestNilPtrValueSub(t *testing.T) {
	var pi *int
	if pv := ValueOf(pi); pv.Elem().IsValid() {
		t.Error("ValueOf((*int)(nil)).Elem().IsValid()")
	}
}

func TestMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	mv := ValueOf(m)
	if n := mv.Len(); n != len(m) {
		t.Errorf("Len = %d, want %d", n, len(m))
	}
	keys := mv.MapKeys()
	newmap := MakeMap(mv.Type())
	for k, v := range m {
		// Check that returned Keys match keys in range.
		// These aren't required to be in the same order.
		seen := false
		for _, kv := range keys {
			if kv.String() == k {
				seen = true
				break
			}
		}
		if !seen {
			t.Errorf("Missing key %q", k)
		}

		// Check that value lookup is correct.
		vv := mv.MapIndex(ValueOf(k))
		if vi := vv.Int(); vi != int64(v) {
			t.Errorf("Key %q: have value %d, want %d", k, vi, v)
		}

		// Copy into new map.
		newmap.SetMapIndex(ValueOf(k), ValueOf(v))
	}
	vv := mv.MapIndex(ValueOf("not-present"))
	if vv.IsValid() {
		t.Errorf("Invalid key: got non-nil value %s", valueToString(vv))
	}

	newm := newmap.Interface().(map[string]int)
	if len(newm) != len(m) {
		t.Errorf("length after copy: newm=%d, m=%d", len(newm), len(m))
	}

	for k, v := range newm {
		mv, ok := m[k]
		if mv != v {
			t.Errorf("newm[%q] = %d, but m[%q] = %d, %v", k, v, k, mv, ok)
		}
	}

	newmap.SetMapIndex(ValueOf("a"), Value{})
	v, ok := newm["a"]
	if ok {
		t.Errorf("newm[\"a\"] = %d after delete", v)
	}

	mv = ValueOf(&m).Elem()
	mv.Set(Zero(mv.Type()))
	if m != nil {
		t.Errorf("mv.Set(nil) failed")
	}
}

func TestNilMap(t *testing.T) {
	var m map[string]int
	mv := ValueOf(m)
	keys := mv.MapKeys()
	if len(keys) != 0 {
		t.Errorf(">0 keys for nil map: %v", keys)
	}

	// Check that value for missing key is zero.
	x := mv.MapIndex(ValueOf("hello"))
	if x.Kind() != Invalid {
		t.Errorf("m.MapIndex(\"hello\") for nil map = %v, want Invalid Value", x)
	}

	// Check big value too.
	var mbig map[string][10 << 20]byte
	x = ValueOf(mbig).MapIndex(ValueOf("hello"))
	if x.Kind() != Invalid {
		t.Errorf("mbig.MapIndex(\"hello\") for nil map = %v, want Invalid Value", x)
	}

	// Test that deletes from a nil map succeed.
	mv.SetMapIndex(ValueOf("hi"), Value{})
}

func TestChan(t *testing.T) {
	for loop := 0; loop < 2; loop++ {
		var c chan int
		var cv Value

		// check both ways to allocate channels
		switch loop {
		case 1:
			c = make(chan int, 1)
			cv = ValueOf(c)
		case 0:
			cv = MakeChan(TypeOf(c), 1)
			c = cv.Interface().(chan int)
		}

		// Send
		cv.Send(ValueOf(2))
		if i := <-c; i != 2 {
			t.Errorf("reflect Send 2, native recv %d", i)
		}

		// Recv
		c <- 3
		if i, ok := cv.Recv(); i.Int() != 3 || !ok {
			t.Errorf("native send 3, reflect Recv %d, %t", i.Int(), ok)
		}

		// TryRecv fail
		val, ok := cv.TryRecv()
		if val.IsValid() || ok {
			t.Errorf("TryRecv on empty chan: %s, %t", valueToString(val), ok)
		}

		// TryRecv success
		c <- 4
		val, ok = cv.TryRecv()
		if !val.IsValid() {
			t.Errorf("TryRecv on ready chan got nil")
		} else if i := val.Int(); i != 4 || !ok {
			t.Errorf("native send 4, TryRecv %d, %t", i, ok)
		}

		// TrySend fail
		c <- 100
		ok = cv.TrySend(ValueOf(5))
		i := <-c
		if ok {
			t.Errorf("TrySend on full chan succeeded: value %d", i)
		}

		// TrySend success
		ok = cv.TrySend(ValueOf(6))
		if !ok {
			t.Errorf("TrySend on empty chan failed")
			select {
			case x := <-c:
				t.Errorf("TrySend failed but it did send %d", x)
			default:
			}
		} else {
			if i = <-c; i != 6 {
				t.Errorf("TrySend 6, recv %d", i)
			}
		}

		// Close
		c <- 123
		cv.Close()
		if i, ok := cv.Recv(); i.Int() != 123 || !ok {
			t.Errorf("send 123 then close; Recv %d, %t", i.Int(), ok)
		}
		if i, ok := cv.Recv(); i.Int() != 0 || ok {
			t.Errorf("after close Recv %d, %t", i.Int(), ok)
		}
	}

	// check creation of unbuffered channel
	var c chan int
	cv := MakeChan(TypeOf(c), 0)
	c = cv.Interface().(chan int)
	if cv.TrySend(ValueOf(7)) {
		t.Errorf("TrySend on sync chan succeeded")
	}
	if v, ok := cv.TryRecv(); v.IsValid() || ok {
		t.Errorf("TryRecv on sync chan succeeded: isvalid=%v ok=%v", v.IsValid(), ok)
	}

	// len/cap
	cv = MakeChan(TypeOf(c), 10)
	c = cv.Interface().(chan int)
	for i := 0; i < 3; i++ {
		c <- i
	}
	if l, m := cv.Len(), cv.Cap(); l != len(c) || m != cap(c) {
		t.Errorf("Len/Cap = %d/%d want %d/%d", l, m, len(c), cap(c))
	}
}

// caseInfo describes a single case in a select test.
type caseInfo struct {
	desc      string
	canSelect bool
	recv      Value
	closed    bool
	helper    func()
	panic     bool
}

var allselect = flag.Bool("allselect", false, "exhaustive select test")

func TestSelect(t *testing.T) {
	selectWatch.once.Do(func() { go selectWatcher() })

	var x exhaustive
	nch := 0
	newop := func(n int, cap int) (ch, val Value) {
		nch++
		if nch%101%2 == 1 {
			c := make(chan int, cap)
			ch = ValueOf(c)
			val = ValueOf(n)
		} else {
			c := make(chan string, cap)
			ch = ValueOf(c)
			val = ValueOf(fmt.Sprint(n))
		}
		return
	}

	for n := 0; x.Next(); n++ {
		if testing.Short() && n >= 1000 {
			break
		}
		if n >= 100000 && !*allselect {
			break
		}
		if n%100000 == 0 && testing.Verbose() {
			println("TestSelect", n)
		}
		var cases []SelectCase
		var info []caseInfo

		// Ready send.
		if x.Maybe() {
			ch, val := newop(len(cases), 1)
			cases = append(cases, SelectCase{
				Dir:  SelectSend,
				Chan: ch,
				Send: val,
			})
			info = append(info, caseInfo{desc: "ready send", canSelect: true})
		}

		// Ready recv.
		if x.Maybe() {
			ch, val := newop(len(cases), 1)
			ch.Send(val)
			cases = append(cases, SelectCase{
				Dir:  SelectRecv,
				Chan: ch,
			})
			info = append(info, caseInfo{desc: "ready recv", canSelect: true, recv: val})
		}

		// Blocking send.
		if x.Maybe() {
			ch, val := newop(len(cases), 0)
			cases = append(cases, SelectCase{
				Dir:  SelectSend,
				Chan: ch,
				Send: val,
			})
			// Let it execute?
			if x.Maybe() {
				f := func() { ch.Recv() }
				info = append(info, caseInfo{desc: "blocking send", helper: f})
			} else {
				info = append(info, caseInfo{desc: "blocking send"})
			}
		}

		// Blocking recv.
		if x.Maybe() {
			ch, val := newop(len(cases), 0)
			cases = append(cases, SelectCase{
				Dir:  SelectRecv,
				Chan: ch,
			})
			// Let it execute?
			if x.Maybe() {
				f := func() { ch.Send(val) }
				info = append(info, caseInfo{desc: "blocking recv", recv: val, helper: f})
			} else {
				info = append(info, caseInfo{desc: "blocking recv"})
			}
		}

		// Zero Chan send.
		if x.Maybe() {
			// Maybe include value to send.
			var val Value
			if x.Maybe() {
				val = ValueOf(100)
			}
			cases = append(cases, SelectCase{
				Dir:  SelectSend,
				Send: val,
			})
			info = append(info, caseInfo{desc: "zero Chan send"})
		}

		// Zero Chan receive.
		if x.Maybe() {
			cases = append(cases, SelectCase{
				Dir: SelectRecv,
			})
			info = append(info, caseInfo{desc: "zero Chan recv"})
		}

		// nil Chan send.
		if x.Maybe() {
			cases = append(cases, SelectCase{
				Dir:  SelectSend,
				Chan: ValueOf((chan int)(nil)),
				Send: ValueOf(101),
			})
			info = append(info, caseInfo{desc: "nil Chan send"})
		}

		// nil Chan recv.
		if x.Maybe() {
			cases = append(cases, SelectCase{
				Dir:  SelectRecv,
				Chan: ValueOf((chan int)(nil)),
			})
			info = append(info, caseInfo{desc: "nil Chan recv"})
		}

		// closed Chan send.
		if x.Maybe() {
			ch := make(chan int)
			close(ch)
			cases = append(cases, SelectCase{
				Dir:  SelectSend,
				Chan: ValueOf(ch),
				Send: ValueOf(101),
			})
			info = append(info, caseInfo{desc: "closed Chan send", canSelect: true, panic: true})
		}

		// closed Chan recv.
		if x.Maybe() {
			ch, val := newop(len(cases), 0)
			ch.Close()
			val = Zero(val.Type())
			cases = append(cases, SelectCase{
				Dir:  SelectRecv,
				Chan: ch,
			})
			info = append(info, caseInfo{desc: "closed Chan recv", canSelect: true, closed: true, recv: val})
		}

		var helper func() // goroutine to help the select complete

		// Add default? Must be last case here, but will permute.
		// Add the default if the select would otherwise
		// block forever, and maybe add it anyway.
		numCanSelect := 0
		canProceed := false
		canBlock := true
		canPanic := false
		helpers := []int{}
		for i, c := range info {
			if c.canSelect {
				canProceed = true
				canBlock = false
				numCanSelect++
				if c.panic {
					canPanic = true
				}
			} else if c.helper != nil {
				canProceed = true
				helpers = append(helpers, i)
			}
		}
		if !canProceed || x.Maybe() {
			cases = append(cases, SelectCase{
				Dir: SelectDefault,
			})
			info = append(info, caseInfo{desc: "default", canSelect: canBlock})
			numCanSelect++
		} else if canBlock {
			// Select needs to communicate with another goroutine.
			cas := &info[helpers[x.Choose(len(helpers))]]
			helper = cas.helper
			cas.canSelect = true
			numCanSelect++
		}

		// Permute cases and case info.
		// Doing too much here makes the exhaustive loop
		// too exhausting, so just do two swaps.
		for loop := 0; loop < 2; loop++ {
			i := x.Choose(len(cases))
			j := x.Choose(len(cases))
			cases[i], cases[j] = cases[j], cases[i]
			info[i], info[j] = info[j], info[i]
		}

		if helper != nil {
			// We wait before kicking off a goroutine to satisfy a blocked select.
			// The pause needs to be big enough to let the select block before
			// we run the helper, but if we lose that race once in a while it's okay: the
			// select will just proceed immediately. Not a big deal.
			// For short tests we can grow [sic] the timeout a bit without fear of taking too long
			pause := 10 * time.Microsecond
			if testing.Short() {
				pause = 100 * time.Microsecond
			}
			time.AfterFunc(pause, helper)
		}

		// Run select.
		i, recv, recvOK, panicErr := runSelect(cases, info)
		if panicErr != nil && !canPanic {
			t.Fatalf("%s\npanicked unexpectedly: %v", fmtSelect(info), panicErr)
		}
		if panicErr == nil && canPanic && numCanSelect == 1 {
			t.Fatalf("%s\nselected #%d incorrectly (should panic)", fmtSelect(info), i)
		}
		if panicErr != nil {
			continue
		}

		cas := info[i]
		if !cas.canSelect {
			recvStr := ""
			if recv.IsValid() {
				recvStr = fmt.Sprintf(", received %v, %v", recv.Interface(), recvOK)
			}
			t.Fatalf("%s\nselected #%d incorrectly%s", fmtSelect(info), i, recvStr)
			continue
		}
		if cas.panic {
			t.Fatalf("%s\nselected #%d incorrectly (case should panic)", fmtSelect(info), i)
			continue
		}

		if cases[i].Dir == SelectRecv {
			if !recv.IsValid() {
				t.Fatalf("%s\nselected #%d but got %v, %v, want %v, %v", fmtSelect(info), i, recv, recvOK, cas.recv.Interface(), !cas.closed)
			}
			if !cas.recv.IsValid() {
				t.Fatalf("%s\nselected #%d but internal error: missing recv value", fmtSelect(info), i)
			}
			if recv.Interface() != cas.recv.Interface() || recvOK != !cas.closed {
				if recv.Interface() == cas.recv.Interface() && recvOK == !cas.closed {
					t.Fatalf("%s\nselected #%d, got %#v, %v, and DeepEqual is broken on %T", fmtSelect(info), i, recv.Interface(), recvOK, recv.Interface())
				}
				t.Fatalf("%s\nselected #%d but got %#v, %v, want %#v, %v", fmtSelect(info), i, recv.Interface(), recvOK, cas.recv.Interface(), !cas.closed)
			}
		} else {
			if recv.IsValid() || recvOK {
				t.Fatalf("%s\nselected #%d but got %v, %v, want %v, %v", fmtSelect(info), i, recv, recvOK, Value{}, false)
			}
		}
	}
}

// selectWatch and the selectWatcher are a watchdog mechanism for running Select.
// If the selectWatcher notices that the select has been blocked for >1 second, it prints
// an error describing the select and panics the entire test binary.
var selectWatch struct {
	sync.Mutex
	once sync.Once
	now  time.Time
	info []caseInfo
}

func selectWatcher() {
	for {
		time.Sleep(1 * time.Second)
		selectWatch.Lock()
		if selectWatch.info != nil && time.Since(selectWatch.now) > 10*time.Second {
			fmt.Fprintf(os.Stderr, "TestSelect:\n%s blocked indefinitely\n", fmtSelect(selectWatch.info))
			panic("select stuck")
		}
		selectWatch.Unlock()
	}
}

// runSelect runs a single select test.
// It returns the values returned by Select but also returns
// a panic value if the Select panics.
func runSelect(cases []SelectCase, info []caseInfo) (chosen int, recv Value, recvOK bool, panicErr any) {
	defer func() {
		panicErr = recover()

		selectWatch.Lock()
		selectWatch.info = nil
		selectWatch.Unlock()
	}()

	selectWatch.Lock()
	selectWatch.now = time.Now()
	selectWatch.info = info
	selectWatch.Unlock()

	chosen, recv, recvOK = Select(cases)
	return
}

// fmtSelect formats the information about a single select test.
func fmtSelect(info []caseInfo) string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "\nselect {\n")
	for i, cas := range info {
		fmt.Fprintf(&buf, "%d: %s", i, cas.desc)
		if cas.recv.IsValid() {
			fmt.Fprintf(&buf, " val=%#v", cas.recv.Interface())
		}
		if cas.canSelect {
			fmt.Fprint(&buf, " canselect")
		}
		if cas.panic {
			fmt.Fprint(&buf, " panic")
		}
		fmt.Fprint(&buf, "\n")
	}
	fmt.Fprint(&buf, "}")
	return buf.String()
}

type two [2]uintptr

// Difficult test for function call because of
// implicit padding between arguments.
func dummy(b byte, c int, d byte, e two, f byte, g float32, h byte) (i byte, j int, k byte, l two, m byte, n float32, o byte) {
	return b, c, d, e, f, g, h
}

func TestFunc(t *testing.T) {
	ret := ValueOf(dummy).Call([]Value{
		ValueOf(byte(10)),
		ValueOf(20),
		ValueOf(byte(30)),
		ValueOf(two{40, 50}),
		ValueOf(byte(60)),
		ValueOf(float32(70)),
		ValueOf(byte(80)),
	})
	if len(ret) != 7 {
		t.Fatalf("Call returned %d values, want 7", len(ret))
	}

	i := byte(ret[0].Uint())
	j := int(ret[1].Int())
	k := byte(ret[2].Uint())
	l := ret[3].Interface().(two)
	m := byte(ret[4].Uint())
	n := float32(ret[5].Float())
	o := byte(ret[6].Uint())

	if i != 10 || j != 20 || k != 30 || l != (two{40, 50}) || m != 60 || n != 70 || o != 80 {
		t.Errorf("Call returned %d, %d, %d, %v, %d, %g, %d; want 10, 20, 30, [40, 50], 60, 70, 80", i, j, k, l, m, n, o)
	}

	for i, v := range ret {
		if v.CanAddr() {
			t.Errorf("result %d is addressable", i)
		}
	}
}

func TestCallConvert(t *testing.T) {
	v := ValueOf(new(io.ReadWriter)).Elem()
	f := ValueOf(func(r io.Reader) io.Reader { return r })
	out := f.Call([]Value{v})
	if len(out) != 1 || out[0].Type() != TypeOf(new(io.Reader)).Elem() || !out[0].IsNil() {
		t.Errorf("expected [nil], got %v", out)
	}
}

type emptyStruct struct{}

type nonEmptyStruct struct {
	member int
}

func returnEmpty() emptyStruct {
	return emptyStruct{}
}

func takesEmpty(e emptyStruct) {}

func returnNonEmpty(i int) nonEmptyStruct {
	return nonEmptyStruct{member: i}
}

func takesNonEmpty(n nonEmptyStruct) int {
	return n.member
}

func TestCallWithStruct(t *testing.T) {
	r := ValueOf(returnEmpty).Call(nil)
	if len(r) != 1 || r[0].Type() != TypeOf(emptyStruct{}) {
		t.Errorf("returning empty struct returned %#v instead", r)
	}
	r = ValueOf(takesEmpty).Call([]Value{ValueOf(emptyStruct{})})
	if len(r) != 0 {
		t.Errorf("takesEmpty returned values: %#v", r)
	}
	r = ValueOf(returnNonEmpty).Call([]Value{ValueOf(42)})
	if len(r) != 1 || r[0].Type() != TypeOf(nonEmptyStruct{}) || r[0].Field(0).Int() != 42 {
		t.Errorf("returnNonEmpty returned %#v", r)
	}
	r = ValueOf(takesNonEmpty).Call([]Value{ValueOf(nonEmptyStruct{member: 42})})
	if len(r) != 1 || r[0].Type() != TypeOf(1) || r[0].Int() != 42 {
		t.Errorf("takesNonEmpty returned %#v", r)
	}
}

func TestCallReturnsEmpty(t *testing.T) {
	// Issue 21717: past-the-end pointer write in Call with
	// nonzero-sized frame and zero-sized return value.
	runtime.GC()
	var finalized uint32
	f := func() (emptyStruct, *[2]int64) {
		i := new([2]int64) // big enough to not be tinyalloc'd, so finalizer always runs when i dies
		runtime.SetFinalizer(i, func(*[2]int64) { atomic.StoreUint32(&finalized, 1) })
		return emptyStruct{}, i
	}
	v := ValueOf(f).Call(nil)[0] // out[0] should not alias out[1]'s memory, so the finalizer should run.
	timeout := time.After(5 * time.Second)
	for atomic.LoadUint32(&finalized) == 0 {
		select {
		case <-timeout:
			t.Fatal("finalizer did not run")
		default:
		}
		runtime.Gosched()
		runtime.GC()
	}
	runtime.KeepAlive(v)
}

func BenchmarkCall(b *testing.B) {
	fv := ValueOf(func(a, b string) {})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		args := []Value{ValueOf("a"), ValueOf("b")}
		for pb.Next() {
			fv.Call(args)
		}
	})
}

func BenchmarkCallArgCopy(b *testing.B) {
	byteArray := func(n int) Value {
		return Zero(ArrayOf(n, TypeOf(byte(0))))
	}
	sizes := [...]struct {
		fv  Value
		arg Value
	}{
		{fv: ValueOf(func(a [128]byte) {}), arg: byteArray(128)},
		{fv: ValueOf(func(a [256]byte) {}), arg: byteArray(256)},
		{fv: ValueOf(func(a [1024]byte) {}), arg: byteArray(1024)},
		{fv: ValueOf(func(a [4096]byte) {}), arg: byteArray(4096)},
		{fv: ValueOf(func(a [65536]byte) {}), arg: byteArray(65536)},
	}
	for _, size := range sizes {
		bench := func(b *testing.B) {
			args := []Value{size.arg}
			b.SetBytes(int64(size.arg.Len()))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					size.fv.Call(args)
				}
			})
		}
		name := fmt.Sprintf("size=%v", size.arg.Len())
		b.Run(name, bench)
	}
}

func TestMakeFunc(t *testing.T) {
	f := dummy
	fv := MakeFunc(TypeOf(f), func(in []Value) []Value { return in })
	ValueOf(&f).Elem().Set(fv)

	// Call g with small arguments so that there is
	// something predictable (and different from the
	// correct results) in those positions on the stack.
	g := dummy
	g(1, 2, 3, two{4, 5}, 6, 7, 8)

	// Call constructed function f.
	i, j, k, l, m, n, o := f(10, 20, 30, two{40, 50}, 60, 70, 80)
	if i != 10 || j != 20 || k != 30 || l != (two{40, 50}) || m != 60 || n != 70 || o != 80 {
		t.Errorf("Call returned %d, %d, %d, %v, %d, %g, %d; want 10, 20, 30, [40, 50], 60, 70, 80", i, j, k, l, m, n, o)
	}
}

func TestMakeFuncInterface(t *testing.T) {
	fn := func(i int) int { return i }
	incr := func(in []Value) []Value {
		return []Value{ValueOf(int(in[0].Int() + 1))}
	}
	fv := MakeFunc(TypeOf(fn), incr)
	ValueOf(&fn).Elem().Set(fv)
	if r := fn(2); r != 3 {
		t.Errorf("Call returned %d, want 3", r)
	}
	if r := fv.Call([]Value{ValueOf(14)})[0].Int(); r != 15 {
		t.Errorf("Call returned %d, want 15", r)
	}
	if r := fv.Interface().(func(int) int)(26); r != 27 {
		t.Errorf("Call returned %d, want 27", r)
	}
}

func TestMakeFuncVariadic(t *testing.T) {
	// Test that variadic arguments are packed into a slice and passed as last arg
	fn := func(_ int, is ...int) []int { return nil }
	fv := MakeFunc(TypeOf(fn), func(in []Value) []Value { return in[1:2] })
	ValueOf(&fn).Elem().Set(fv)

	r := fn(1, 2, 3)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}

	r = fn(1, []int{2, 3}...)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}

	r = fv.Call([]Value{ValueOf(1), ValueOf(2), ValueOf(3)})[0].Interface().([]int)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}

	r = fv.CallSlice([]Value{ValueOf(1), ValueOf([]int{2, 3})})[0].Interface().([]int)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}

	f := fv.Interface().(func(int, ...int) []int)

	r = f(1, 2, 3)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}
	r = f(1, []int{2, 3}...)
	if r[0] != 2 || r[1] != 3 {
		t.Errorf("Call returned [%v, %v]; want 2, 3", r[0], r[1])
	}
}

type Point struct {
	x, y int
}

// This will be index 0.
func (p Point) AnotherMethod(scale int) int {
	return -1
}

// This will be index 1.
func (p Point) Dist(scale int) int {
	// println("Point.Dist", p.x, p.y, scale)
	return p.x*p.x*scale + p.y*p.y*scale
}

// This will be index 2.
func (p Point) GCMethod(k int) int {
	runtime.GC()
	return k + p.x
}

// This will be index 3.
func (p Point) NoArgs() {
	// Exercise no-argument/no-result paths.
}

// This will be index 4.
func (p Point) TotalDist(points ...Point) int {
	tot := 0
	for _, q := range points {
		dx := q.x - p.x
		dy := q.y - p.y
		tot += dx*dx + dy*dy // Should call Sqrt, but it's just a test.
	}
	return tot
}

func TestMethod(t *testing.T) {
	// Non-curried method of type.
	p := Point{x: 3, y: 4}
	i := TypeOf(p).Method(1).Func.Call([]Value{ValueOf(p), ValueOf(10)})[0].Int()
	if i != 250 {
		t.Errorf("Type Method returned %d; want 250", i)
	}

	m, ok := TypeOf(p).MethodByName("Dist")
	if !ok {
		t.Fatalf("method by name failed")
	}
	i = m.Func.Call([]Value{ValueOf(p), ValueOf(11)})[0].Int()
	if i != 275 {
		t.Errorf("Type MethodByName returned %d; want 275", i)
	}

	m, ok = TypeOf(p).MethodByName("NoArgs")
	if !ok {
		t.Fatalf("method by name failed")
	}
	n := len(m.Func.Call([]Value{ValueOf(p)}))
	if n != 0 {
		t.Errorf("NoArgs returned %d values; want 0", n)
	}

	i = TypeOf(&p).Method(1).Func.Call([]Value{ValueOf(&p), ValueOf(12)})[0].Int()
	if i != 300 {
		t.Errorf("Pointer Type Method returned %d; want 300", i)
	}

	m, ok = TypeOf(&p).MethodByName("Dist")
	if !ok {
		t.Fatalf("ptr method by name failed")
	}
	i = m.Func.Call([]Value{ValueOf(&p), ValueOf(13)})[0].Int()
	if i != 325 {
		t.Errorf("Pointer Type MethodByName returned %d; want 325", i)
	}

	m, ok = TypeOf(&p).MethodByName("NoArgs")
	if !ok {
		t.Fatalf("method by name failed")
	}
	n = len(m.Func.Call([]Value{ValueOf(&p)}))
	if n != 0 {
		t.Errorf("NoArgs returned %d values; want 0", n)
	}

	// Curried method of value.
	tfunc := TypeOf((func(int) int)(nil))
	v := ValueOf(p).Method(1)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Value Method Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(14)})[0].Int()
	if i != 350 {
		t.Errorf("Value Method returned %d; want 350", i)
	}
	v = ValueOf(p).MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Value MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(15)})[0].Int()
	if i != 375 {
		t.Errorf("Value MethodByName returned %d; want 375", i)
	}
	v = ValueOf(p).MethodByName("NoArgs")
	v.Call(nil)

	// Curried method of pointer.
	v = ValueOf(&p).Method(1)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Value Method Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(16)})[0].Int()
	if i != 400 {
		t.Errorf("Pointer Value Method returned %d; want 400", i)
	}
	v = ValueOf(&p).MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Value MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(17)})[0].Int()
	if i != 425 {
		t.Errorf("Pointer Value MethodByName returned %d; want 425", i)
	}
	v = ValueOf(&p).MethodByName("NoArgs")
	v.Call(nil)

	// Curried method of interface value.
	// Have to wrap interface value in a struct to get at it.
	// Passing it to ValueOf directly would
	// access the underlying Point, not the interface.
	var x interface {
		Dist(int) int
	} = p
	pv := ValueOf(&x).Elem()
	v = pv.Method(0)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Interface Method Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(18)})[0].Int()
	if i != 450 {
		t.Errorf("Interface Method returned %d; want 450", i)
	}
	v = pv.MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Interface MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = v.Call([]Value{ValueOf(19)})[0].Int()
	if i != 475 {
		t.Errorf("Interface MethodByName returned %d; want 475", i)
	}
}

func TestMethodValue(t *testing.T) {
	p := Point{x: 3, y: 4}
	var i int64

	// Curried method of value.
	tfunc := TypeOf((func(int) int)(nil))
	v := ValueOf(p).Method(1)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Value Method Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(10)})[0].Int()
	if i != 250 {
		t.Errorf("Value Method returned %d; want 250", i)
	}
	v = ValueOf(p).MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Value MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(11)})[0].Int()
	if i != 275 {
		t.Errorf("Value MethodByName returned %d; want 275", i)
	}
	v = ValueOf(p).MethodByName("NoArgs")
	ValueOf(v.Interface()).Call(nil)
	v.Interface().(func())()

	// Curried method of pointer.
	v = ValueOf(&p).Method(1)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Value Method Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(12)})[0].Int()
	if i != 300 {
		t.Errorf("Pointer Value Method returned %d; want 300", i)
	}
	v = ValueOf(&p).MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Value MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(13)})[0].Int()
	if i != 325 {
		t.Errorf("Pointer Value MethodByName returned %d; want 325", i)
	}
	v = ValueOf(&p).MethodByName("NoArgs")
	ValueOf(v.Interface()).Call(nil)
	v.Interface().(func())()

	// Curried method of pointer to pointer.
	pp := &p
	v = ValueOf(&pp).Elem().Method(1)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Pointer Value Method Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(14)})[0].Int()
	if i != 350 {
		t.Errorf("Pointer Pointer Value Method returned %d; want 350", i)
	}
	v = ValueOf(&pp).Elem().MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Pointer Pointer Value MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(15)})[0].Int()
	if i != 375 {
		t.Errorf("Pointer Pointer Value MethodByName returned %d; want 375", i)
	}

	// Curried method of interface value.
	// Have to wrap interface value in a struct to get at it.
	// Passing it to ValueOf directly would
	// access the underlying Point, not the interface.
	var s = struct {
		X interface {
			Dist(int) int
		}
	}{X: p}
	pv := ValueOf(s).Field(0)
	v = pv.Method(0)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Interface Method Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(16)})[0].Int()
	if i != 400 {
		t.Errorf("Interface Method returned %d; want 400", i)
	}
	v = pv.MethodByName("Dist")
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Interface MethodByName Type is %s; want %s", tt, tfunc)
	}
	i = ValueOf(v.Interface()).Call([]Value{ValueOf(17)})[0].Int()
	if i != 425 {
		t.Errorf("Interface MethodByName returned %d; want 425", i)
	}
}

func TestVariadicMethodValue(t *testing.T) {
	p := Point{x: 3, y: 4}
	points := []Point{{x: 20, y: 21}, {x: 22, y: 23}, {x: 24, y: 25}}
	want := int64(p.TotalDist(points[0], points[1], points[2]))

	// Curried method of value.
	tfunc := TypeOf((func(...Point) int)(nil))
	v := ValueOf(p).Method(4)
	if tt := v.Type(); tt != tfunc {
		t.Errorf("Variadic Method Type is %s; want %s", tt, tfunc)
	}
	i := ValueOf(v.Interface()).Call([]Value{ValueOf(points[0]), ValueOf(points[1]), ValueOf(points[2])})[0].Int()
	if i != want {
		t.Errorf("Variadic Method returned %d; want %d", i, want)
	}
	i = ValueOf(v.Interface()).CallSlice([]Value{ValueOf(points)})[0].Int()
	if i != want {
		t.Errorf("Variadic Method CallSlice returned %d; want %d", i, want)
	}

	f := v.Interface().(func(...Point) int)
	i = int64(f(points[0], points[1], points[2]))
	if i != want {
		t.Errorf("Variadic Method Interface returned %d; want %d", i, want)
	}
	i = int64(f(points...))
	if i != want {
		t.Errorf("Variadic Method Interface Slice returned %d; want %d", i, want)
	}
}

// Reflect version of $GOROOT/test/method5.go

// Concrete types implementing M method.
// Smaller than a word, word-sized, larger than a word.
// Value and pointer receivers.

type Tinter interface {
	M(int, byte) (byte, int)
}

type Tsmallv byte

func (v Tsmallv) M(x int, b byte) (byte, int) { return b, x + int(v) }

type Tsmallp byte

func (p *Tsmallp) M(x int, b byte) (byte, int) { return b, x + int(*p) }

type Twordv uintptr

func (v Twordv) M(x int, b byte) (byte, int) { return b, x + int(v) }

type Twordp uintptr

func (p *Twordp) M(x int, b byte) (byte, int) { return b, x + int(*p) }

type Tbigv [2]uintptr

func (v Tbigv) M(x int, b byte) (byte, int) { return b, x + int(v[0]) + int(v[1]) }

type Tbigp [2]uintptr

func (p *Tbigp) M(x int, b byte) (byte, int) { return b, x + int(p[0]) + int(p[1]) }

type tinter interface {
	m(int, byte) (byte, int)
}

// Embedding via pointer.

type Tm1 struct {
	Tm2
}

type Tm2 struct {
	*Tm3
}

type Tm3 struct {
	*Tm4
}

type Tm4 struct {
}

func (t4 Tm4) M(x int, b byte) (byte, int) { return b, x + 40 }

func TestMethod5(t *testing.T) {
	CheckF := func(name string, f func(int, byte) (byte, int), inc int) {
		b, x := f(1000, 99)
		if b != 99 || x != 1000+inc {
			t.Errorf("%s(1000, 99) = %v, %v, want 99, %v", name, b, x, 1000+inc)
		}
	}

	CheckV := func(name string, i Value, inc int) {
		bx := i.Method(0).Call([]Value{ValueOf(1000), ValueOf(byte(99))})
		b := bx[0].Interface()
		x := bx[1].Interface()
		if b != byte(99) || x != 1000+inc {
			t.Errorf("direct %s.M(1000, 99) = %v, %v, want 99, %v", name, b, x, 1000+inc)
		}

		CheckF(name+".M", i.Method(0).Interface().(func(int, byte) (byte, int)), inc)
	}

	var TinterType = TypeOf(new(Tinter)).Elem()

	CheckI := func(name string, i any, inc int) {
		v := ValueOf(i)
		CheckV(name, v, inc)
		CheckV("(i="+name+")", v.Convert(TinterType), inc)
	}

	sv := Tsmallv(1)
	CheckI("sv", sv, 1)
	CheckI("&sv", &sv, 1)

	sp := Tsmallp(2)
	CheckI("&sp", &sp, 2)

	wv := Twordv(3)
	CheckI("wv", wv, 3)
	CheckI("&wv", &wv, 3)

	wp := Twordp(4)
	CheckI("&wp", &wp, 4)

	bv := Tbigv([2]uintptr{5, 6})
	CheckI("bv", bv, 11)
	CheckI("&bv", &bv, 11)

	bp := Tbigp([2]uintptr{7, 8})
	CheckI("&bp", &bp, 15)

	t4 := Tm4{}
	t3 := Tm3{Tm4: &t4}
	t2 := Tm2{Tm3: &t3}
	t1 := Tm1{Tm2: t2}
	CheckI("t4", t4, 40)
	CheckI("&t4", &t4, 40)
	CheckI("t3", t3, 40)
	CheckI("&t3", &t3, 40)
	CheckI("t2", t2, 40)
	CheckI("&t2", &t2, 40)
	CheckI("t1", t1, 40)
	CheckI("&t1", &t1, 40)

	var tnil Tinter
	vnil := ValueOf(&tnil).Elem()
	shouldPanic(func() { vnil.Method(0) })
}

func TestInterfaceSet(t *testing.T) {
	p := &Point{x: 3, y: 4}

	var s struct {
		I any
		P interface {
			Dist(int) int
		}
	}
	sv := ValueOf(&s).Elem()
	sv.Field(0).Set(ValueOf(p))
	if q := s.I.(*Point); q != p {
		t.Errorf("i: have %p want %p", q, p)
	}

	pv := sv.Field(1)
	pv.Set(ValueOf(p))
	if q := s.P.(*Point); q != p {
		t.Errorf("i: have %p want %p", q, p)
	}

	i := pv.Method(0).Call([]Value{ValueOf(10)})[0].Int()
	if i != 250 {
		t.Errorf("Interface Method returned %d; want 250", i)
	}
}

type T1 struct {
	a string
	int
}

func TestAnonymousFields(t *testing.T) {
	var field StructField
	var ok bool
	var t1 T1
	type1 := TypeOf(t1)
	if field, ok = type1.FieldByName("int"); !ok {
		t.Fatal("no field 'int'")
	}
	if field.Index[0] != 1 {
		t.Error("field index should be 1; is", field.Index)
	}
}

type FTest struct {
	s     any
	name  string
	index []int
	value int
}

type D1 struct {
	d int
}

type D2 struct {
	d int
}

type S0 struct {
	A, B, C int
	D1
	D2
}

type S1 struct {
	B int
	S0
}

type S2 struct {
	A int
	*S1
}

type S1x struct {
	S1
}

type S1y struct {
	S1
}

type S3 struct {
	S1x
	S2
	D, E int
	*S1y
}

type S4 struct {
	*S4
	A int
}

// The X in S6 and S7 annihilate, but they also block the X in S8.S9.
type S5 struct {
	S6
	S7
	S8
}

type S6 struct {
	X int
}

type S7 S6

type S8 struct {
	S9
}

type S9 struct {
	X int
	Y int
}

// The X in S11.S6 and S12.S6 annihilate, but they also block the X in S13.S8.S9.
type S10 struct {
	S11
	S12
	S13
}

type S11 struct {
	S6
}

type S12 struct {
	S6
}

type S13 struct {
	S8
}

// The X in S15.S11.S1 and S16.S11.S1 annihilate.
type S14 struct {
	S15
	S16
}

type S15 struct {
	S11
}

type S16 struct {
	S11
}

var fieldTests = []FTest{
	{s: struct{}{}, name: "", index: nil, value: 0},
	{s: struct{}{}, name: "Foo", index: nil, value: 0},
	{s: S0{A: 'a'}, name: "A", index: []int{0}, value: 'a'},
	{s: S0{}, name: "D", index: nil, value: 0},
	{s: S1{S0: S0{A: 'a'}}, name: "A", index: []int{1, 0}, value: 'a'},
	{s: S1{B: 'b'}, name: "B", index: []int{0}, value: 'b'},
	{s: S1{}, name: "S0", index: []int{1}, value: 0},
	{s: S1{S0: S0{C: 'c'}}, name: "C", index: []int{1, 2}, value: 'c'},
	{s: S2{A: 'a'}, name: "A", index: []int{0}, value: 'a'},
	{s: S2{}, name: "S1", index: []int{1}, value: 0},
	{s: S2{S1: &S1{B: 'b'}}, name: "B", index: []int{1, 0}, value: 'b'},
	{s: S2{S1: &S1{S0: S0{C: 'c'}}}, name: "C", index: []int{1, 1, 2}, value: 'c'},
	{s: S2{}, name: "D", index: nil, value: 0},
	{s: S3{}, name: "S1", index: nil, value: 0},
	{s: S3{S2: S2{A: 'a'}}, name: "A", index: []int{1, 0}, value: 'a'},
	{s: S3{}, name: "B", index: nil, value: 0},
	{s: S3{D: 'd'}, name: "D", index: []int{2}, value: 0},
	{s: S3{E: 'e'}, name: "E", index: []int{3}, value: 'e'},
	{s: S4{A: 'a'}, name: "A", index: []int{1}, value: 'a'},
	{s: S4{}, name: "B", index: nil, value: 0},
	{s: S5{}, name: "X", index: nil, value: 0},
	{s: S5{}, name: "Y", index: []int{2, 0, 1}, value: 0},
	{s: S10{}, name: "X", index: nil, value: 0},
	{s: S10{}, name: "Y", index: []int{2, 0, 0, 1}, value: 0},
	{s: S14{}, name: "X", index: nil, value: 0},
}

func TestFieldByIndex(t *testing.T) {
	for _, test := range fieldTests {
		s := TypeOf(test.s)
		f := s.FieldByIndex(test.index)
		if f.Name != "" {
			if test.index != nil {
				if f.Name != test.name {
					t.Errorf("%s.%s found; want %s", s.Name(), f.Name, test.name)
				}
			} else {
				t.Errorf("%s.%s found", s.Name(), f.Name)
			}
		} else if len(test.index) > 0 {
			t.Errorf("%s.%s not found", s.Name(), test.name)
		}

		if test.value != 0 {
			v := ValueOf(test.s).FieldByIndex(test.index)
			if v.IsValid() {
				if x, ok := v.Interface().(int); ok {
					if x != test.value {
						t.Errorf("%s%v is %d; want %d", s.Name(), test.index, x, test.value)
					}
				} else {
					t.Errorf("%s%v value not an int", s.Name(), test.index)
				}
			} else {
				t.Errorf("%s%v value not found", s.Name(), test.index)
			}
		}
	}
}

func TestFieldByName(t *testing.T) {
	for _, test := range fieldTests {
		s := TypeOf(test.s)
		f, found := s.FieldByName(test.name)
		if found {
			if test.index != nil {
				// Verify field depth and index.
				if len(f.Index) != len(test.index) {
					t.Errorf("%s.%s depth %d; want %d: %v vs %v", s.Name(), test.name, len(f.Index), len(test.index), f.Index, test.index)
				} else {
					for i, x := range f.Index {
						if x != test.index[i] {
							t.Errorf("%s.%s.Index[%d] is %d; want %d", s.Name(), test.name, i, x, test.index[i])
						}
					}
				}
			} else {
				t.Errorf("%s.%s found", s.Name(), f.Name)
			}
		} else if len(test.index) > 0 {
			t.Errorf("%s.%s not found", s.Name(), test.name)
		}

		if test.value != 0 {
			v := ValueOf(test.s).FieldByName(test.name)
			if v.IsValid() {
				if x, ok := v.Interface().(int); ok {
					if x != test.value {
						t.Errorf("%s.%s is %d; want %d", s.Name(), test.name, x, test.value)
					}
				} else {
					t.Errorf("%s.%s value not an int", s.Name(), test.name)
				}
			} else {
				t.Errorf("%s.%s value not found", s.Name(), test.name)
			}
		}
	}
}

func TestImportPath(t *testing.T) {
	tests := []struct {
		t    Type
		path string
	}{
		{t: TypeOf(&base64.Encoding{}).Elem(), path: "encoding/base64"},
		{t: TypeOf(int(0)), path: ""},
		{t: TypeOf(int8(0)), path: ""},
		{t: TypeOf(int16(0)), path: ""},
		{t: TypeOf(int32(0)), path: ""},
		{t: TypeOf(int64(0)), path: ""},
		{t: TypeOf(uint(0)), path: ""},
		{t: TypeOf(uint8(0)), path: ""},
		{t: TypeOf(uint16(0)), path: ""},
		{t: TypeOf(uint32(0)), path: ""},
		{t: TypeOf(uint64(0)), path: ""},
		{t: TypeOf(uintptr(0)), path: ""},
		{t: TypeOf(float32(0)), path: ""},
		{t: TypeOf(float64(0)), path: ""},
		{t: TypeOf(complex64(0)), path: ""},
		{t: TypeOf(complex128(0)), path: ""},
		{t: TypeOf(byte(0)), path: ""},
		{t: TypeOf(rune(0)), path: ""},
		{t: TypeOf([]byte(nil)), path: ""},
		{t: TypeOf([]rune(nil)), path: ""},
		{t: TypeOf(string("")), path: ""},
		{t: TypeOf((*any)(nil)).Elem(), path: ""},
		{t: TypeOf((*byte)(nil)), path: ""},
		{t: TypeOf((*rune)(nil)), path: ""},
		{t: TypeOf((*int64)(nil)), path: ""},
		{t: TypeOf(map[string]int{}), path: ""},
		{t: TypeOf((*error)(nil)).Elem(), path: ""},
		{t: TypeOf((*Point)(nil)), path: ""},
		{t: TypeOf((*Point)(nil)).Elem(), path: "github.com/goccy/go-reflect_test"},
	}
	for _, test := range tests {
		if path := test.t.PkgPath(); path != test.path {
			t.Errorf("%v.PkgPath() = %q, want %q", test.t, path, test.path)
		}
	}
}

func TestVariadicType(t *testing.T) {
	// Test example from Type documentation.
	var f func(x int, y ...float64)
	typ := TypeOf(f)
	if typ.NumIn() == 2 && typ.In(0) == TypeOf(int(0)) {
		sl := typ.In(1)
		if sl.Kind() == Slice {
			if sl.Elem() == TypeOf(0.0) {
				// ok
				return
			}
		}
	}

	// Failed
	t.Errorf("want NumIn() = 2, In(0) = int, In(1) = []float64")
	s := fmt.Sprintf("have NumIn() = %d", typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		s += fmt.Sprintf(", In(%d) = %s", i, typ.In(i))
	}
	t.Error(s)
}

type inner struct {
	x int
}

type outer struct {
	y int
	inner
}

func (*inner) M() {}

func (*outer) M() {}

func TestNestedMethods(t *testing.T) {
	typ := TypeOf((*outer)(nil))
	if typ.NumMethod() != 1 || typ.Method(0).Func.Pointer() != ValueOf((*outer).M).Pointer() {
		t.Errorf("Wrong method table for outer: (M=%p)", (*outer).M)
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			t.Errorf("\t%d: %s %#x\n", i, m.Name, m.Func.Pointer())
		}
	}
}

type unexp struct{}

func (*unexp) f() (int32, int8) { return 7, 7 }

func (*unexp) g() (int64, int8) { return 8, 8 }

type unexpI interface {
	f() (int32, int8)
}

var unexpi unexpI = new(unexp)

func TestUnexportedMethods(t *testing.T) {
	typ := TypeOf(unexpi)

	if got := typ.NumMethod(); got != 0 {
		t.Errorf("NumMethod=%d, want 0 satisfied methods", got)
	}
}

type InnerInt struct {
	X int
}

type OuterInt struct {
	Y int
	InnerInt
}

func (i *InnerInt) M() int {
	return i.X
}

func TestEmbeddedMethods(t *testing.T) {
	typ := TypeOf((*OuterInt)(nil))
	if typ.NumMethod() != 1 || typ.Method(0).Func.Pointer() != ValueOf((*OuterInt).M).Pointer() {
		t.Errorf("Wrong method table for OuterInt: (m=%p)", (*OuterInt).M)
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			t.Errorf("\t%d: %s %#x\n", i, m.Name, m.Func.Pointer())
		}
	}

	i := &InnerInt{X: 3}
	if v := ValueOf(i).Method(0).Call(nil)[0].Int(); v != 3 {
		t.Errorf("i.M() = %d, want 3", v)
	}

	o := &OuterInt{Y: 1, InnerInt: InnerInt{X: 2}}
	if v := ValueOf(o).Method(0).Call(nil)[0].Int(); v != 2 {
		t.Errorf("i.M() = %d, want 2", v)
	}

	f := (*OuterInt).M
	if v := f(o); v != 2 {
		t.Errorf("f(o) = %d, want 2", v)
	}
}

type FuncDDD func(...any) error

func (f FuncDDD) M() {}

func TestNumMethodOnDDD(t *testing.T) {
	rv := ValueOf((FuncDDD)(nil))
	if n := rv.NumMethod(); n != 1 {
		t.Fatalf("NumMethod()=%d, want 1", n)
	}
}

func TestPtrTo(t *testing.T) {
	// This block of code means that the ptrToThis field of the
	// reflect data for *unsafe.Pointer is non zero, see
	// https://golang.org/issue/19003
	var x unsafe.Pointer
	var y = &x
	var z = &y

	var i int

	typ := TypeOf(z)
	for i = 0; i < 100; i++ {
		typ = PtrTo(typ)
	}
	for i = 0; i < 100; i++ {
		typ = typ.Elem()
	}
	if typ != TypeOf(z) {
		t.Errorf("after 100 PtrTo and Elem, have %s, want %s", typ, TypeOf(z))
	}
}

func TestPtrToGC(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	pt := PtrTo(tt)
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := New(pt)
		p := new(*uintptr)
		*p = new(uintptr)
		**p = uintptr(i)
		v.Elem().Set(ValueOf(p).Convert(pt))
		x = append(x, v.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		k := ValueOf(xi).Elem().Elem().Elem().Interface().(uintptr)
		if k != uintptr(i) {
			t.Errorf("lost x[%d] = %d, want %d", i, k, i)
		}
	}
}

func BenchmarkPtrTo(b *testing.B) {
	// Construct a type with a zero ptrToThis.
	type T struct{ int }
	t := SliceOf(TypeOf(T{}))
	ptrToThis := ValueOf(t).Elem().FieldByName("ptrToThis")
	if !ptrToThis.IsValid() {
		b.Fatalf("%v has no ptrToThis field; was it removed from rtype?", t)
	}
	if ptrToThis.Int() != 0 {
		b.Fatalf("%v.ptrToThis unexpectedly nonzero", t)
	}
	b.ResetTimer()

	// Now benchmark calling PtrTo on it: we'll have to hit the ptrMap cache on
	// every call.
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			PtrTo(t)
		}
	})
}

func TestAddr(t *testing.T) {
	var p struct {
		X, Y int
	}

	v := ValueOf(&p)
	v = v.Elem()
	v = v.Addr()
	v = v.Elem()
	v = v.Field(0)
	v.SetInt(2)
	if p.X != 2 {
		t.Errorf("Addr.Elem.Set failed to set value")
	}

	// Again but take address of the ValueOf value.
	// Exercises generation of PtrTypes not present in the binary.
	q := &p
	v = ValueOf(&q).Elem()
	v = v.Addr()
	v = v.Elem()
	v = v.Elem()
	v = v.Addr()
	v = v.Elem()
	v = v.Field(0)
	v.SetInt(3)
	if p.X != 3 {
		t.Errorf("Addr.Elem.Set failed to set value")
	}

	// Starting without pointer we should get changed value
	// in interface.
	qq := p
	v = ValueOf(&qq).Elem()
	v0 := v
	v = v.Addr()
	v = v.Elem()
	v = v.Field(0)
	v.SetInt(4)
	if p.X != 3 { // should be unchanged from last time
		t.Errorf("somehow value Set changed original p")
	}
	p = v0.Interface().(struct {
		X, Y int
	})
	if p.X != 4 {
		t.Errorf("Addr.Elem.Set valued to set value in top value")
	}

	// Verify that taking the address of a type gives us a pointer
	// which we can convert back using the usual interface
	// notation.
	var s struct {
		B *bool
	}
	ps := ValueOf(&s).Elem().Field(0).Addr().Interface()
	*(ps.(**bool)) = new(bool)
	if s.B == nil {
		t.Errorf("Addr.Interface direct assignment failed")
	}
}

func noAlloc(t *testing.T, n int, f func(int)) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Skip("skipping; GOMAXPROCS>1")
	}
	i := -1
	allocs := testing.AllocsPerRun(n, func() {
		f(i)
		i++
	})
	if allocs > 0 {
		t.Errorf("%d iterations: got %v mallocs, want 0", n, allocs)
	}
}

func TestAllocations(t *testing.T) {
	noAlloc(t, 100, func(j int) {
		var i any
		var v Value

		// We can uncomment this when compiler escape analysis
		// is good enough to see that the integer assigned to i
		// does not escape and therefore need not be allocated.
		//
		// i = 42 + j
		// v = ValueOf(i)
		// if int(v.Int()) != 42+j {
		// 	panic("wrong int")
		// }

		i = func(j int) int { return j }
		v = ValueOf(i)
		if v.Interface().(func(int) int)(j) != j {
			panic("wrong result")
		}
	})
}

func TestSmallNegativeInt(t *testing.T) {
	i := int16(-1)
	v := ValueOf(i)
	if v.Int() != -1 {
		t.Errorf("int16(-1).Int() returned %v", v.Int())
	}
}

func TestIndex(t *testing.T) {
	xs := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	v := ValueOf(xs).Index(3).Interface().(byte)
	if v != xs[3] {
		t.Errorf("xs.Index(3) = %v; expected %v", v, xs[3])
	}
	xa := [8]byte{10, 20, 30, 40, 50, 60, 70, 80}
	v = ValueOf(xa).Index(2).Interface().(byte)
	if v != xa[2] {
		t.Errorf("xa.Index(2) = %v; expected %v", v, xa[2])
	}
	s := "0123456789"
	v = ValueOf(s).Index(3).Interface().(byte)
	if v != s[3] {
		t.Errorf("s.Index(3) = %v; expected %v", v, s[3])
	}
}

func TestSlice(t *testing.T) {
	xs := []int{1, 2, 3, 4, 5, 6, 7, 8}
	v := ValueOf(xs).Slice(3, 5).Interface().([]int)
	if len(v) != 2 {
		t.Errorf("len(xs.Slice(3, 5)) = %d", len(v))
	}
	if cap(v) != 5 {
		t.Errorf("cap(xs.Slice(3, 5)) = %d", cap(v))
	}
	if !DeepEqual(v[0:5], xs[3:]) {
		t.Errorf("xs.Slice(3, 5)[0:5] = %v", v[0:5])
	}
	xa := [8]int{10, 20, 30, 40, 50, 60, 70, 80}
	v = ValueOf(&xa).Elem().Slice(2, 5).Interface().([]int)
	if len(v) != 3 {
		t.Errorf("len(xa.Slice(2, 5)) = %d", len(v))
	}
	if cap(v) != 6 {
		t.Errorf("cap(xa.Slice(2, 5)) = %d", cap(v))
	}
	if !DeepEqual(v[0:6], xa[2:]) {
		t.Errorf("xs.Slice(2, 5)[0:6] = %v", v[0:6])
	}
	s := "0123456789"
	vs := ValueOf(s).Slice(3, 5).Interface().(string)
	if vs != s[3:5] {
		t.Errorf("s.Slice(3, 5) = %q; expected %q", vs, s[3:5])
	}

	rv := ValueOf(&xs).Elem()
	rv = rv.Slice(3, 4)
	ptr2 := rv.Pointer()
	rv = rv.Slice(5, 5)
	ptr3 := rv.Pointer()
	if ptr3 != ptr2 {
		t.Errorf("xs.Slice(3,4).Slice3(5,5).Pointer() = %#x, want %#x", ptr3, ptr2)
	}
}

func TestSlice3(t *testing.T) {
	xs := []int{1, 2, 3, 4, 5, 6, 7, 8}
	v := ValueOf(xs).Slice3(3, 5, 7).Interface().([]int)
	if len(v) != 2 {
		t.Errorf("len(xs.Slice3(3, 5, 7)) = %d", len(v))
	}
	if cap(v) != 4 {
		t.Errorf("cap(xs.Slice3(3, 5, 7)) = %d", cap(v))
	}
	if !DeepEqual(v[0:4], xs[3:7:7]) {
		t.Errorf("xs.Slice3(3, 5, 7)[0:4] = %v", v[0:4])
	}
	rv := ValueOf(&xs).Elem()
	shouldPanic(func() { rv.Slice3(1, 2, 1) })
	shouldPanic(func() { rv.Slice3(1, 1, 11) })
	shouldPanic(func() { rv.Slice3(2, 2, 1) })

	xa := [8]int{10, 20, 30, 40, 50, 60, 70, 80}
	v = ValueOf(&xa).Elem().Slice3(2, 5, 6).Interface().([]int)
	if len(v) != 3 {
		t.Errorf("len(xa.Slice(2, 5, 6)) = %d", len(v))
	}
	if cap(v) != 4 {
		t.Errorf("cap(xa.Slice(2, 5, 6)) = %d", cap(v))
	}
	if !DeepEqual(v[0:4], xa[2:6:6]) {
		t.Errorf("xs.Slice(2, 5, 6)[0:4] = %v", v[0:4])
	}
	rv = ValueOf(&xa).Elem()
	shouldPanic(func() { rv.Slice3(1, 2, 1) })
	shouldPanic(func() { rv.Slice3(1, 1, 11) })
	shouldPanic(func() { rv.Slice3(2, 2, 1) })

	s := "hello world"
	rv = ValueOf(&s).Elem()
	shouldPanic(func() { rv.Slice3(1, 2, 3) })

	rv = ValueOf(&xs).Elem()
	rv = rv.Slice3(3, 5, 7)
	ptr2 := rv.Pointer()
	rv = rv.Slice3(4, 4, 4)
	ptr3 := rv.Pointer()
	if ptr3 != ptr2 {
		t.Errorf("xs.Slice3(3,5,7).Slice3(4,4,4).Pointer() = %#x, want %#x", ptr3, ptr2)
	}
}

func TestSetLenCap(t *testing.T) {
	xs := []int{1, 2, 3, 4, 5, 6, 7, 8}
	xa := [8]int{10, 20, 30, 40, 50, 60, 70, 80}

	vs := ValueOf(&xs).Elem()
	shouldPanic(func() { vs.SetLen(10) })
	shouldPanic(func() { vs.SetCap(10) })
	shouldPanic(func() { vs.SetLen(-1) })
	shouldPanic(func() { vs.SetCap(-1) })
	shouldPanic(func() { vs.SetCap(6) }) // smaller than len
	vs.SetLen(5)
	if len(xs) != 5 || cap(xs) != 8 {
		t.Errorf("after SetLen(5), len, cap = %d, %d, want 5, 8", len(xs), cap(xs))
	}
	vs.SetCap(6)
	if len(xs) != 5 || cap(xs) != 6 {
		t.Errorf("after SetCap(6), len, cap = %d, %d, want 5, 6", len(xs), cap(xs))
	}
	vs.SetCap(5)
	if len(xs) != 5 || cap(xs) != 5 {
		t.Errorf("after SetCap(5), len, cap = %d, %d, want 5, 5", len(xs), cap(xs))
	}
	shouldPanic(func() { vs.SetCap(4) }) // smaller than len
	shouldPanic(func() { vs.SetLen(6) }) // bigger than cap

	va := ValueOf(&xa).Elem()
	shouldPanic(func() { va.SetLen(8) })
	shouldPanic(func() { va.SetCap(8) })
}

func TestVariadic(t *testing.T) {
	var b bytes.Buffer
	V := ValueOf

	b.Reset()
	V(fmt.Fprintf).Call([]Value{V(&b), V("%s, %d world"), V("hello"), V(42)})
	if b.String() != "hello, 42 world" {
		t.Errorf("after Fprintf Call: %q != %q", b.String(), "hello 42 world")
	}

	b.Reset()
	V(fmt.Fprintf).CallSlice([]Value{V(&b), V("%s, %d world"), V([]any{"hello", 42})})
	if b.String() != "hello, 42 world" {
		t.Errorf("after Fprintf CallSlice: %q != %q", b.String(), "hello 42 world")
	}
}

func TestFuncArg(t *testing.T) {
	f1 := func(i int, f func(int) int) int { return f(i) }
	f2 := func(i int) int { return i + 1 }
	r := ValueOf(f1).Call([]Value{ValueOf(100), ValueOf(f2)})
	if r[0].Int() != 101 {
		t.Errorf("function returned %d, want 101", r[0].Int())
	}
}

func TestStructArg(t *testing.T) {
	type padded struct {
		B string
		C int32
	}
	var (
		gotA  padded
		gotB  uint32
		wantA = padded{B: "3", C: 4}
		wantB = uint32(5)
	)
	f := func(a padded, b uint32) {
		gotA, gotB = a, b
	}
	ValueOf(f).Call([]Value{ValueOf(wantA), ValueOf(wantB)})
	if gotA != wantA || gotB != wantB {
		t.Errorf("function called with (%v, %v), want (%v, %v)", gotA, gotB, wantA, wantB)
	}
}

var tagGetTests = []struct {
	Tag   StructTag
	Key   string
	Value string
}{
	{Tag: `protobuf:"PB(1,2)"`, Key: `protobuf`, Value: `PB(1,2)`},
	{Tag: `protobuf:"PB(1,2)"`, Key: `foo`, Value: ``},
	{Tag: `protobuf:"PB(1,2)"`, Key: `rotobuf`, Value: ``},
	{Tag: `protobuf:"PB(1,2)" json:"name"`, Key: `json`, Value: `name`},
	{Tag: `protobuf:"PB(1,2)" json:"name"`, Key: `protobuf`, Value: `PB(1,2)`},
	{Tag: `k0:"values contain spaces" k1:"and\ttabs"`, Key: "k0", Value: "values contain spaces"},
	{Tag: `k0:"values contain spaces" k1:"and\ttabs"`, Key: "k1", Value: "and\ttabs"},
}

func TestTagGet(t *testing.T) {
	for _, tt := range tagGetTests {
		if v := tt.Tag.Get(tt.Key); v != tt.Value {
			t.Errorf("StructTag(%#q).Get(%#q) = %#q, want %#q", tt.Tag, tt.Key, v, tt.Value)
		}
	}
}

func TestBytes(t *testing.T) {
	type B []byte
	x := B{1, 2, 3, 4}
	y := ValueOf(x).Bytes()
	if !bytes.Equal(x, y) {
		t.Fatalf("ValueOf(%v).Bytes() = %v", x, y)
	}
	if &x[0] != &y[0] {
		t.Errorf("ValueOf(%p).Bytes() = %p", &x[0], &y[0])
	}
}

func TestSetBytes(t *testing.T) {
	type B []byte
	var x B
	y := []byte{1, 2, 3, 4}
	ValueOf(&x).Elem().SetBytes(y)
	if !bytes.Equal(x, y) {
		t.Fatalf("ValueOf(%v).Bytes() = %v", x, y)
	}
	if &x[0] != &y[0] {
		t.Errorf("ValueOf(%p).Bytes() = %p", &x[0], &y[0])
	}
}

type Private struct {
	x int
	y **int
	Z int
}

func (p *Private) m() {}

type private struct {
	Z int
	z int
	S string
	A [1]Private
	T []Private
}

func (p *private) P() {}

type Public struct {
	X int
	Y **int
	private
}

func (p *Public) M() {}

func TestUnexported(t *testing.T) {
	var pub Public
	pub.S = "S"
	pub.T = pub.A[:]
	v := ValueOf(&pub)
	isValid(v.Elem().Field(0))
	isValid(v.Elem().Field(1))
	isValid(v.Elem().Field(2))
	isValid(v.Elem().FieldByName("X"))
	isValid(v.Elem().FieldByName("Y"))
	isValid(v.Elem().FieldByName("Z"))
	isValid(v.Type().Method(0).Func)
	m, _ := v.Type().MethodByName("M")
	isValid(m.Func)
	m, _ = v.Type().MethodByName("P")
	isValid(m.Func)
	isNonNil(v.Elem().Field(0).Interface())
	isNonNil(v.Elem().Field(1).Interface())
	isNonNil(v.Elem().Field(2).Field(2).Index(0))
	isNonNil(v.Elem().FieldByName("X").Interface())
	isNonNil(v.Elem().FieldByName("Y").Interface())
	isNonNil(v.Elem().FieldByName("Z").Interface())
	isNonNil(v.Elem().FieldByName("S").Index(0).Interface())
	isNonNil(v.Type().Method(0).Func.Interface())
	m, _ = v.Type().MethodByName("P")
	isNonNil(m.Func.Interface())

	var priv Private
	v = ValueOf(&priv)
	isValid(v.Elem().Field(0))
	isValid(v.Elem().Field(1))
	isValid(v.Elem().FieldByName("x"))
	isValid(v.Elem().FieldByName("y"))
	shouldPanic(func() { v.Elem().Field(0).Interface() })
	shouldPanic(func() { v.Elem().Field(1).Interface() })
	shouldPanic(func() { v.Elem().FieldByName("x").Interface() })
	shouldPanic(func() { v.Elem().FieldByName("y").Interface() })
	shouldPanic(func() { v.Type().Method(0) })
}

func TestSetPanic(t *testing.T) {
	ok := func(f func()) { f() }
	bad := shouldPanic
	clear := func(v Value) { v.Set(Zero(v.Type())) }

	type t0 struct {
		W int
	}

	type t1 struct {
		Y int
		t0
	}

	type T2 struct {
		Z       int
		namedT0 t0
	}

	type T struct {
		X int
		t1
		T2
		NamedT1 t1
		NamedT2 T2
		namedT1 t1
		namedT2 T2
	}

	// not addressable
	v := ValueOf(T{})
	bad(func() { clear(v.Field(0)) })                   // .X
	bad(func() { clear(v.Field(1)) })                   // .t1
	bad(func() { clear(v.Field(1).Field(0)) })          // .t1.Y
	bad(func() { clear(v.Field(1).Field(1)) })          // .t1.t0
	bad(func() { clear(v.Field(1).Field(1).Field(0)) }) // .t1.t0.W
	bad(func() { clear(v.Field(2)) })                   // .T2
	bad(func() { clear(v.Field(2).Field(0)) })          // .T2.Z
	bad(func() { clear(v.Field(2).Field(1)) })          // .T2.namedT0
	bad(func() { clear(v.Field(2).Field(1).Field(0)) }) // .T2.namedT0.W
	bad(func() { clear(v.Field(3)) })                   // .NamedT1
	bad(func() { clear(v.Field(3).Field(0)) })          // .NamedT1.Y
	bad(func() { clear(v.Field(3).Field(1)) })          // .NamedT1.t0
	bad(func() { clear(v.Field(3).Field(1).Field(0)) }) // .NamedT1.t0.W
	bad(func() { clear(v.Field(4)) })                   // .NamedT2
	bad(func() { clear(v.Field(4).Field(0)) })          // .NamedT2.Z
	bad(func() { clear(v.Field(4).Field(1)) })          // .NamedT2.namedT0
	bad(func() { clear(v.Field(4).Field(1).Field(0)) }) // .NamedT2.namedT0.W
	bad(func() { clear(v.Field(5)) })                   // .namedT1
	bad(func() { clear(v.Field(5).Field(0)) })          // .namedT1.Y
	bad(func() { clear(v.Field(5).Field(1)) })          // .namedT1.t0
	bad(func() { clear(v.Field(5).Field(1).Field(0)) }) // .namedT1.t0.W
	bad(func() { clear(v.Field(6)) })                   // .namedT2
	bad(func() { clear(v.Field(6).Field(0)) })          // .namedT2.Z
	bad(func() { clear(v.Field(6).Field(1)) })          // .namedT2.namedT0
	bad(func() { clear(v.Field(6).Field(1).Field(0)) }) // .namedT2.namedT0.W

	// addressable
	v = ValueOf(&T{}).Elem()
	ok(func() { clear(v.Field(0)) })                    // .X
	bad(func() { clear(v.Field(1)) })                   // .t1
	ok(func() { clear(v.Field(1).Field(0)) })           // .t1.Y
	bad(func() { clear(v.Field(1).Field(1)) })          // .t1.t0
	ok(func() { clear(v.Field(1).Field(1).Field(0)) })  // .t1.t0.W
	ok(func() { clear(v.Field(2)) })                    // .T2
	ok(func() { clear(v.Field(2).Field(0)) })           // .T2.Z
	bad(func() { clear(v.Field(2).Field(1)) })          // .T2.namedT0
	bad(func() { clear(v.Field(2).Field(1).Field(0)) }) // .T2.namedT0.W
	ok(func() { clear(v.Field(3)) })                    // .NamedT1
	ok(func() { clear(v.Field(3).Field(0)) })           // .NamedT1.Y
	bad(func() { clear(v.Field(3).Field(1)) })          // .NamedT1.t0
	ok(func() { clear(v.Field(3).Field(1).Field(0)) })  // .NamedT1.t0.W
	ok(func() { clear(v.Field(4)) })                    // .NamedT2
	ok(func() { clear(v.Field(4).Field(0)) })           // .NamedT2.Z
	bad(func() { clear(v.Field(4).Field(1)) })          // .NamedT2.namedT0
	bad(func() { clear(v.Field(4).Field(1).Field(0)) }) // .NamedT2.namedT0.W
	bad(func() { clear(v.Field(5)) })                   // .namedT1
	bad(func() { clear(v.Field(5).Field(0)) })          // .namedT1.Y
	bad(func() { clear(v.Field(5).Field(1)) })          // .namedT1.t0
	bad(func() { clear(v.Field(5).Field(1).Field(0)) }) // .namedT1.t0.W
	bad(func() { clear(v.Field(6)) })                   // .namedT2
	bad(func() { clear(v.Field(6).Field(0)) })          // .namedT2.Z
	bad(func() { clear(v.Field(6).Field(1)) })          // .namedT2.namedT0
	bad(func() { clear(v.Field(6).Field(1).Field(0)) }) // .namedT2.namedT0.W
}

type timp int

func (t timp) W() {}

func (t timp) Y() {}

func (t timp) w() {}

func (t timp) y() {}

func TestCallPanic(t *testing.T) {
	type t0 interface {
		W()
		w()
	}
	type T1 interface {
		Y()
		y()
	}
	type T2 struct {
		T1
		t0
	}
	type T struct {
		t0 // 0
		T1 // 1

		NamedT0 t0 // 2
		NamedT1 T1 // 3
		NamedT2 T2 // 4

		namedT0 t0 // 5
		namedT1 T1 // 6
		namedT2 T2 // 7
	}
	ok := func(f func()) { f() }
	bad := shouldPanic
	call := func(v Value) { v.Call(nil) }

	i := timp(0)
	v := ValueOf(T{t0: i, T1: i, NamedT0: i, NamedT1: i, NamedT2: T2{T1: i, t0: i}, namedT0: i, namedT1: i, namedT2: T2{T1: i, t0: i}})
	// ok(func() { call(v.Field(0).Method(0)) })         // .t0.W
	// bad(func() { call(v.Field(0).Elem().Method(0)) }) // .t0.W
	// bad(func() { call(v.Field(0).Method(1)) })        // .t0.w
	// bad(func() { call(v.Field(0).Elem().Method(2)) }) // .t0.w
	ok(func() { call(v.Field(1).Method(0)) })         // .T1.Y
	ok(func() { call(v.Field(1).Elem().Method(0)) })  // .T1.Y
	bad(func() { call(v.Field(1).Method(1)) })        // .T1.y
	bad(func() { call(v.Field(1).Elem().Method(2)) }) // .T1.y

	ok(func() { call(v.Field(2).Method(0)) })         // .NamedT0.W
	ok(func() { call(v.Field(2).Elem().Method(0)) })  // .NamedT0.W
	bad(func() { call(v.Field(2).Method(1)) })        // .NamedT0.w
	bad(func() { call(v.Field(2).Elem().Method(2)) }) // .NamedT0.w

	ok(func() { call(v.Field(3).Method(0)) })         // .NamedT1.Y
	ok(func() { call(v.Field(3).Elem().Method(0)) })  // .NamedT1.Y
	bad(func() { call(v.Field(3).Method(1)) })        // .NamedT1.y
	bad(func() { call(v.Field(3).Elem().Method(3)) }) // .NamedT1.y

	ok(func() { call(v.Field(4).Field(0).Method(0)) })        // .NamedT2.T1.Y
	ok(func() { call(v.Field(4).Field(0).Elem().Method(0)) }) // .NamedT2.T1.W
	// ok(func() { call(v.Field(4).Field(1).Method(0)) })         // .NamedT2.t0.W
	// bad(func() { call(v.Field(4).Field(1).Elem().Method(0)) }) // .NamedT2.t0.W

	bad(func() { call(v.Field(5).Method(0)) })        // .namedT0.W
	bad(func() { call(v.Field(5).Elem().Method(0)) }) // .namedT0.W
	bad(func() { call(v.Field(5).Method(1)) })        // .namedT0.w
	bad(func() { call(v.Field(5).Elem().Method(2)) }) // .namedT0.w

	bad(func() { call(v.Field(6).Method(0)) })        // .namedT1.Y
	bad(func() { call(v.Field(6).Elem().Method(0)) }) // .namedT1.Y
	bad(func() { call(v.Field(6).Method(0)) })        // .namedT1.y
	bad(func() { call(v.Field(6).Elem().Method(0)) }) // .namedT1.y

	bad(func() { call(v.Field(7).Field(0).Method(0)) })        // .namedT2.T1.Y
	bad(func() { call(v.Field(7).Field(0).Elem().Method(0)) }) // .namedT2.T1.W
	// bad(func() { call(v.Field(7).Field(1).Method(0)) })        // .namedT2.t0.W
	// bad(func() { call(v.Field(7).Field(1).Elem().Method(0)) }) // .namedT2.t0.W
}

func shouldPanic(f func()) {
	defer func() {
		if recover() == nil {
			panic("did not panic")
		}
	}()
	f()
}

func isNonNil(x any) {
	if x == nil {
		panic("nil interface")
	}
}

func isValid(v Value) {
	if !v.IsValid() {
		panic("zero Value")
	}
}

func TestAlias(t *testing.T) {
	x := string("hello")
	v := ValueOf(&x).Elem()
	oldvalue := v.Interface()
	v.SetString("world")
	newvalue := v.Interface()

	if oldvalue != "hello" || newvalue != "world" {
		t.Errorf("aliasing: old=%q new=%q, want hello, world", oldvalue, newvalue)
	}
}

var V = ValueOf

func EmptyInterfaceV(x any) Value {
	return ValueOf(&x).Elem()
}

func ReaderV(x io.Reader) Value {
	return ValueOf(&x).Elem()
}

func ReadWriterV(x io.ReadWriter) Value {
	return ValueOf(&x).Elem()
}

type Empty struct{}

type MyStruct struct {
	x int `some:"tag"`
}

type MyString string

type MyBytes []byte

type MyRunes []int32

type MyFunc func()

type MyByte byte

var convertTests = []struct {
	in  Value
	out Value
}{
	// numbers
	/*
		Edit .+1,/\*\//-1>cat >/tmp/x.go && go run /tmp/x.go

		package main

		import "fmt"

		var numbers = []string{
			"int8", "uint8", "int16", "uint16",
			"int32", "uint32", "int64", "uint64",
			"int", "uint", "uintptr",
			"float32", "float64",
		}

		func main() {
			// all pairs but in an unusual order,
			// to emit all the int8, uint8 cases
			// before n grows too big.
			n := 1
			for i, f := range numbers {
				for _, g := range numbers[i:] {
					fmt.Printf("\t{V(%s(%d)), V(%s(%d))},\n", f, n, g, n)
					n++
					if f != g {
						fmt.Printf("\t{V(%s(%d)), V(%s(%d))},\n", g, n, f, n)
						n++
					}
				}
			}
		}
	*/
	{in: V(int8(1)), out: V(int8(1))},
	{in: V(int8(2)), out: V(uint8(2))},
	{in: V(uint8(3)), out: V(int8(3))},
	{in: V(int8(4)), out: V(int16(4))},
	{in: V(int16(5)), out: V(int8(5))},
	{in: V(int8(6)), out: V(uint16(6))},
	{in: V(uint16(7)), out: V(int8(7))},
	{in: V(int8(8)), out: V(int32(8))},
	{in: V(int32(9)), out: V(int8(9))},
	{in: V(int8(10)), out: V(uint32(10))},
	{in: V(uint32(11)), out: V(int8(11))},
	{in: V(int8(12)), out: V(int64(12))},
	{in: V(int64(13)), out: V(int8(13))},
	{in: V(int8(14)), out: V(uint64(14))},
	{in: V(uint64(15)), out: V(int8(15))},
	{in: V(int8(16)), out: V(int(16))},
	{in: V(int(17)), out: V(int8(17))},
	{in: V(int8(18)), out: V(uint(18))},
	{in: V(uint(19)), out: V(int8(19))},
	{in: V(int8(20)), out: V(uintptr(20))},
	{in: V(uintptr(21)), out: V(int8(21))},
	{in: V(int8(22)), out: V(float32(22))},
	{in: V(float32(23)), out: V(int8(23))},
	{in: V(int8(24)), out: V(float64(24))},
	{in: V(float64(25)), out: V(int8(25))},
	{in: V(uint8(26)), out: V(uint8(26))},
	{in: V(uint8(27)), out: V(int16(27))},
	{in: V(int16(28)), out: V(uint8(28))},
	{in: V(uint8(29)), out: V(uint16(29))},
	{in: V(uint16(30)), out: V(uint8(30))},
	{in: V(uint8(31)), out: V(int32(31))},
	{in: V(int32(32)), out: V(uint8(32))},
	{in: V(uint8(33)), out: V(uint32(33))},
	{in: V(uint32(34)), out: V(uint8(34))},
	{in: V(uint8(35)), out: V(int64(35))},
	{in: V(int64(36)), out: V(uint8(36))},
	{in: V(uint8(37)), out: V(uint64(37))},
	{in: V(uint64(38)), out: V(uint8(38))},
	{in: V(uint8(39)), out: V(int(39))},
	{in: V(int(40)), out: V(uint8(40))},
	{in: V(uint8(41)), out: V(uint(41))},
	{in: V(uint(42)), out: V(uint8(42))},
	{in: V(uint8(43)), out: V(uintptr(43))},
	{in: V(uintptr(44)), out: V(uint8(44))},
	{in: V(uint8(45)), out: V(float32(45))},
	{in: V(float32(46)), out: V(uint8(46))},
	{in: V(uint8(47)), out: V(float64(47))},
	{in: V(float64(48)), out: V(uint8(48))},
	{in: V(int16(49)), out: V(int16(49))},
	{in: V(int16(50)), out: V(uint16(50))},
	{in: V(uint16(51)), out: V(int16(51))},
	{in: V(int16(52)), out: V(int32(52))},
	{in: V(int32(53)), out: V(int16(53))},
	{in: V(int16(54)), out: V(uint32(54))},
	{in: V(uint32(55)), out: V(int16(55))},
	{in: V(int16(56)), out: V(int64(56))},
	{in: V(int64(57)), out: V(int16(57))},
	{in: V(int16(58)), out: V(uint64(58))},
	{in: V(uint64(59)), out: V(int16(59))},
	{in: V(int16(60)), out: V(int(60))},
	{in: V(int(61)), out: V(int16(61))},
	{in: V(int16(62)), out: V(uint(62))},
	{in: V(uint(63)), out: V(int16(63))},
	{in: V(int16(64)), out: V(uintptr(64))},
	{in: V(uintptr(65)), out: V(int16(65))},
	{in: V(int16(66)), out: V(float32(66))},
	{in: V(float32(67)), out: V(int16(67))},
	{in: V(int16(68)), out: V(float64(68))},
	{in: V(float64(69)), out: V(int16(69))},
	{in: V(uint16(70)), out: V(uint16(70))},
	{in: V(uint16(71)), out: V(int32(71))},
	{in: V(int32(72)), out: V(uint16(72))},
	{in: V(uint16(73)), out: V(uint32(73))},
	{in: V(uint32(74)), out: V(uint16(74))},
	{in: V(uint16(75)), out: V(int64(75))},
	{in: V(int64(76)), out: V(uint16(76))},
	{in: V(uint16(77)), out: V(uint64(77))},
	{in: V(uint64(78)), out: V(uint16(78))},
	{in: V(uint16(79)), out: V(int(79))},
	{in: V(int(80)), out: V(uint16(80))},
	{in: V(uint16(81)), out: V(uint(81))},
	{in: V(uint(82)), out: V(uint16(82))},
	{in: V(uint16(83)), out: V(uintptr(83))},
	{in: V(uintptr(84)), out: V(uint16(84))},
	{in: V(uint16(85)), out: V(float32(85))},
	{in: V(float32(86)), out: V(uint16(86))},
	{in: V(uint16(87)), out: V(float64(87))},
	{in: V(float64(88)), out: V(uint16(88))},
	{in: V(int32(89)), out: V(int32(89))},
	{in: V(int32(90)), out: V(uint32(90))},
	{in: V(uint32(91)), out: V(int32(91))},
	{in: V(int32(92)), out: V(int64(92))},
	{in: V(int64(93)), out: V(int32(93))},
	{in: V(int32(94)), out: V(uint64(94))},
	{in: V(uint64(95)), out: V(int32(95))},
	{in: V(int32(96)), out: V(int(96))},
	{in: V(int(97)), out: V(int32(97))},
	{in: V(int32(98)), out: V(uint(98))},
	{in: V(uint(99)), out: V(int32(99))},
	{in: V(int32(100)), out: V(uintptr(100))},
	{in: V(uintptr(101)), out: V(int32(101))},
	{in: V(int32(102)), out: V(float32(102))},
	{in: V(float32(103)), out: V(int32(103))},
	{in: V(int32(104)), out: V(float64(104))},
	{in: V(float64(105)), out: V(int32(105))},
	{in: V(uint32(106)), out: V(uint32(106))},
	{in: V(uint32(107)), out: V(int64(107))},
	{in: V(int64(108)), out: V(uint32(108))},
	{in: V(uint32(109)), out: V(uint64(109))},
	{in: V(uint64(110)), out: V(uint32(110))},
	{in: V(uint32(111)), out: V(int(111))},
	{in: V(int(112)), out: V(uint32(112))},
	{in: V(uint32(113)), out: V(uint(113))},
	{in: V(uint(114)), out: V(uint32(114))},
	{in: V(uint32(115)), out: V(uintptr(115))},
	{in: V(uintptr(116)), out: V(uint32(116))},
	{in: V(uint32(117)), out: V(float32(117))},
	{in: V(float32(118)), out: V(uint32(118))},
	{in: V(uint32(119)), out: V(float64(119))},
	{in: V(float64(120)), out: V(uint32(120))},
	{in: V(int64(121)), out: V(int64(121))},
	{in: V(int64(122)), out: V(uint64(122))},
	{in: V(uint64(123)), out: V(int64(123))},
	{in: V(int64(124)), out: V(int(124))},
	{in: V(int(125)), out: V(int64(125))},
	{in: V(int64(126)), out: V(uint(126))},
	{in: V(uint(127)), out: V(int64(127))},
	{in: V(int64(128)), out: V(uintptr(128))},
	{in: V(uintptr(129)), out: V(int64(129))},
	{in: V(int64(130)), out: V(float32(130))},
	{in: V(float32(131)), out: V(int64(131))},
	{in: V(int64(132)), out: V(float64(132))},
	{in: V(float64(133)), out: V(int64(133))},
	{in: V(uint64(134)), out: V(uint64(134))},
	{in: V(uint64(135)), out: V(int(135))},
	{in: V(int(136)), out: V(uint64(136))},
	{in: V(uint64(137)), out: V(uint(137))},
	{in: V(uint(138)), out: V(uint64(138))},
	{in: V(uint64(139)), out: V(uintptr(139))},
	{in: V(uintptr(140)), out: V(uint64(140))},
	{in: V(uint64(141)), out: V(float32(141))},
	{in: V(float32(142)), out: V(uint64(142))},
	{in: V(uint64(143)), out: V(float64(143))},
	{in: V(float64(144)), out: V(uint64(144))},
	{in: V(int(145)), out: V(int(145))},
	{in: V(int(146)), out: V(uint(146))},
	{in: V(uint(147)), out: V(int(147))},
	{in: V(int(148)), out: V(uintptr(148))},
	{in: V(uintptr(149)), out: V(int(149))},
	{in: V(int(150)), out: V(float32(150))},
	{in: V(float32(151)), out: V(int(151))},
	{in: V(int(152)), out: V(float64(152))},
	{in: V(float64(153)), out: V(int(153))},
	{in: V(uint(154)), out: V(uint(154))},
	{in: V(uint(155)), out: V(uintptr(155))},
	{in: V(uintptr(156)), out: V(uint(156))},
	{in: V(uint(157)), out: V(float32(157))},
	{in: V(float32(158)), out: V(uint(158))},
	{in: V(uint(159)), out: V(float64(159))},
	{in: V(float64(160)), out: V(uint(160))},
	{in: V(uintptr(161)), out: V(uintptr(161))},
	{in: V(uintptr(162)), out: V(float32(162))},
	{in: V(float32(163)), out: V(uintptr(163))},
	{in: V(uintptr(164)), out: V(float64(164))},
	{in: V(float64(165)), out: V(uintptr(165))},
	{in: V(float32(166)), out: V(float32(166))},
	{in: V(float32(167)), out: V(float64(167))},
	{in: V(float64(168)), out: V(float32(168))},
	{in: V(float64(169)), out: V(float64(169))},

	// truncation
	{in: V(float64(1.5)), out: V(int(1))},

	// complex
	{in: V(complex64(1i)), out: V(complex64(1i))},
	{in: V(complex64(2i)), out: V(complex128(2i))},
	{in: V(complex128(3i)), out: V(complex64(3i))},
	{in: V(complex128(4i)), out: V(complex128(4i))},

	// string
	{in: V(string("hello")), out: V(string("hello"))},
	{in: V(string("bytes1")), out: V([]byte("bytes1"))},
	{in: V([]byte("bytes2")), out: V(string("bytes2"))},
	{in: V([]byte("bytes3")), out: V([]byte("bytes3"))},
	{in: V(string("runes♝")), out: V([]rune("runes♝"))},
	{in: V([]rune("runes♕")), out: V(string("runes♕"))},
	{in: V([]rune("runes🙈🙉🙊")), out: V([]rune("runes🙈🙉🙊"))},
	{in: V(int('a')), out: V(string("a"))},
	{in: V(int8('a')), out: V(string("a"))},
	{in: V(int16('a')), out: V(string("a"))},
	{in: V(int32('a')), out: V(string("a"))},
	{in: V(int64('a')), out: V(string("a"))},
	{in: V(uint('a')), out: V(string("a"))},
	{in: V(uint8('a')), out: V(string("a"))},
	{in: V(uint16('a')), out: V(string("a"))},
	{in: V(uint32('a')), out: V(string("a"))},
	{in: V(uint64('a')), out: V(string("a"))},
	{in: V(uintptr('a')), out: V(string("a"))},
	{in: V(int(-1)), out: V(string("\uFFFD"))},
	{in: V(int8(-2)), out: V(string("\uFFFD"))},
	{in: V(int16(-3)), out: V(string("\uFFFD"))},
	{in: V(int32(-4)), out: V(string("\uFFFD"))},
	{in: V(int64(-5)), out: V(string("\uFFFD"))},
	{in: V(uint(0x110001)), out: V(string("\uFFFD"))},
	{in: V(uint32(0x110002)), out: V(string("\uFFFD"))},
	{in: V(uint64(0x110003)), out: V(string("\uFFFD"))},
	{in: V(uintptr(0x110004)), out: V(string("\uFFFD"))},

	// named string
	{in: V(MyString("hello")), out: V(string("hello"))},
	{in: V(string("hello")), out: V(MyString("hello"))},
	{in: V(string("hello")), out: V(string("hello"))},
	{in: V(MyString("hello")), out: V(MyString("hello"))},
	{in: V(MyString("bytes1")), out: V([]byte("bytes1"))},
	{in: V([]byte("bytes2")), out: V(MyString("bytes2"))},
	{in: V([]byte("bytes3")), out: V([]byte("bytes3"))},
	{in: V(MyString("runes♝")), out: V([]rune("runes♝"))},
	{in: V([]rune("runes♕")), out: V(MyString("runes♕"))},
	{in: V([]rune("runes🙈🙉🙊")), out: V([]rune("runes🙈🙉🙊"))},
	{in: V([]rune("runes🙈🙉🙊")), out: V(MyRunes("runes🙈🙉🙊"))},
	{in: V(MyRunes("runes🙈🙉🙊")), out: V([]rune("runes🙈🙉🙊"))},
	{in: V(int('a')), out: V(MyString("a"))},
	{in: V(int8('a')), out: V(MyString("a"))},
	{in: V(int16('a')), out: V(MyString("a"))},
	{in: V(int32('a')), out: V(MyString("a"))},
	{in: V(int64('a')), out: V(MyString("a"))},
	{in: V(uint('a')), out: V(MyString("a"))},
	{in: V(uint8('a')), out: V(MyString("a"))},
	{in: V(uint16('a')), out: V(MyString("a"))},
	{in: V(uint32('a')), out: V(MyString("a"))},
	{in: V(uint64('a')), out: V(MyString("a"))},
	{in: V(uintptr('a')), out: V(MyString("a"))},
	{in: V(int(-1)), out: V(MyString("\uFFFD"))},
	{in: V(int8(-2)), out: V(MyString("\uFFFD"))},
	{in: V(int16(-3)), out: V(MyString("\uFFFD"))},
	{in: V(int32(-4)), out: V(MyString("\uFFFD"))},
	{in: V(int64(-5)), out: V(MyString("\uFFFD"))},
	{in: V(uint(0x110001)), out: V(MyString("\uFFFD"))},
	{in: V(uint32(0x110002)), out: V(MyString("\uFFFD"))},
	{in: V(uint64(0x110003)), out: V(MyString("\uFFFD"))},
	{in: V(uintptr(0x110004)), out: V(MyString("\uFFFD"))},

	// named []byte
	{in: V(string("bytes1")), out: V(MyBytes("bytes1"))},
	{in: V(MyBytes("bytes2")), out: V(string("bytes2"))},
	{in: V(MyBytes("bytes3")), out: V(MyBytes("bytes3"))},
	{in: V(MyString("bytes1")), out: V(MyBytes("bytes1"))},
	{in: V(MyBytes("bytes2")), out: V(MyString("bytes2"))},

	// named []rune
	{in: V(string("runes♝")), out: V(MyRunes("runes♝"))},
	{in: V(MyRunes("runes♕")), out: V(string("runes♕"))},
	{in: V(MyRunes("runes🙈🙉🙊")), out: V(MyRunes("runes🙈🙉🙊"))},
	{in: V(MyString("runes♝")), out: V(MyRunes("runes♝"))},
	{in: V(MyRunes("runes♕")), out: V(MyString("runes♕"))},

	// named types and equal underlying types
	{in: V(new(int)), out: V(new(integer))},
	{in: V(new(integer)), out: V(new(int))},
	{in: V(Empty{}), out: V(struct{}{})},
	{in: V(new(Empty)), out: V(new(struct{}))},
	{in: V(struct{}{}), out: V(Empty{})},
	{in: V(new(struct{})), out: V(new(Empty))},
	{in: V(Empty{}), out: V(Empty{})},
	{in: V(MyBytes{}), out: V([]byte{})},
	{in: V([]byte{}), out: V(MyBytes{})},
	{in: V((func())(nil)), out: V(MyFunc(nil))},
	{in: V((MyFunc)(nil)), out: V((func())(nil))},

	// structs with different tags
	{in: V(struct {
		x int `some:"foo"`
	}{}), out: V(struct {
		x int `some:"bar"`
	}{})},

	{in: V(struct {
		x int `some:"bar"`
	}{}), out: V(struct {
		x int `some:"foo"`
	}{})},

	{in: V(MyStruct{}), out: V(struct {
		x int `some:"foo"`
	}{})},

	{in: V(struct {
		x int `some:"foo"`
	}{}), out: V(MyStruct{})},

	{in: V(MyStruct{}), out: V(struct {
		x int `some:"bar"`
	}{})},

	{in: V(struct {
		x int `some:"bar"`
	}{}), out: V(MyStruct{})},

	// can convert *byte and *MyByte
	{in: V((*byte)(nil)), out: V((*MyByte)(nil))},
	{in: V((*MyByte)(nil)), out: V((*byte)(nil))},

	// cannot convert mismatched array sizes
	{in: V([2]byte{}), out: V([2]byte{})},
	{in: V([3]byte{}), out: V([3]byte{})},

	// cannot convert other instances
	{in: V((**byte)(nil)), out: V((**byte)(nil))},
	{in: V((**MyByte)(nil)), out: V((**MyByte)(nil))},
	{in: V((chan byte)(nil)), out: V((chan byte)(nil))},
	{in: V((chan MyByte)(nil)), out: V((chan MyByte)(nil))},
	{in: V(([]byte)(nil)), out: V(([]byte)(nil))},
	{in: V(([]MyByte)(nil)), out: V(([]MyByte)(nil))},
	{in: V((map[int]byte)(nil)), out: V((map[int]byte)(nil))},
	{in: V((map[int]MyByte)(nil)), out: V((map[int]MyByte)(nil))},
	{in: V((map[byte]int)(nil)), out: V((map[byte]int)(nil))},
	{in: V((map[MyByte]int)(nil)), out: V((map[MyByte]int)(nil))},
	{in: V([2]byte{}), out: V([2]byte{})},
	{in: V([2]MyByte{}), out: V([2]MyByte{})},

	// other
	{in: V((***int)(nil)), out: V((***int)(nil))},
	{in: V((***byte)(nil)), out: V((***byte)(nil))},
	{in: V((***int32)(nil)), out: V((***int32)(nil))},
	{in: V((***int64)(nil)), out: V((***int64)(nil))},
	{in: V((chan int)(nil)), out: V((<-chan int)(nil))},
	{in: V((chan int)(nil)), out: V((chan<- int)(nil))},
	{in: V((chan string)(nil)), out: V((<-chan string)(nil))},
	{in: V((chan string)(nil)), out: V((chan<- string)(nil))},
	{in: V((chan byte)(nil)), out: V((chan byte)(nil))},
	{in: V((chan MyByte)(nil)), out: V((chan MyByte)(nil))},
	{in: V((map[int]bool)(nil)), out: V((map[int]bool)(nil))},
	{in: V((map[int]byte)(nil)), out: V((map[int]byte)(nil))},
	{in: V((map[uint]bool)(nil)), out: V((map[uint]bool)(nil))},
	{in: V([]uint(nil)), out: V([]uint(nil))},
	{in: V([]int(nil)), out: V([]int(nil))},
	{in: V(new(any)), out: V(new(any))},
	{in: V(new(io.Reader)), out: V(new(io.Reader))},
	{in: V(new(io.Writer)), out: V(new(io.Writer))},

	// interfaces
	{in: V(int(1)), out: EmptyInterfaceV(int(1))},
	{in: V(string("hello")), out: EmptyInterfaceV(string("hello"))},
	{in: V(new(bytes.Buffer)), out: ReaderV(new(bytes.Buffer))},
	{in: ReadWriterV(new(bytes.Buffer)), out: ReaderV(new(bytes.Buffer))},
	{in: V(new(bytes.Buffer)), out: ReadWriterV(new(bytes.Buffer))},
}

func TestConvert(t *testing.T) {
	canConvert := map[[2]Type]bool{}
	all := map[Type]bool{}

	for _, tt := range convertTests {
		t1 := tt.in.Type()
		if !t1.ConvertibleTo(t1) {
			t.Errorf("(%s).ConvertibleTo(%s) = false, want true", t1, t1)
			continue
		}

		t2 := tt.out.Type()
		if !t1.ConvertibleTo(t2) {
			t.Errorf("(%s).ConvertibleTo(%s) = false, want true", t1, t2)
			continue
		}

		all[t1] = true
		all[t2] = true
		canConvert[[2]Type{t1, t2}] = true

		// vout1 represents the in value converted to the in type.
		v1 := tt.in
		vout1 := v1.Convert(t1)
		out1 := vout1.Interface()
		if vout1.Type() != tt.in.Type() || !DeepEqual(out1, tt.in.Interface()) {
			t.Errorf("ValueOf(%T(%[1]v)).Convert(%s) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t1, out1, tt.in.Interface())
		}

		// vout2 represents the in value converted to the out type.
		vout2 := v1.Convert(t2)
		out2 := vout2.Interface()
		if vout2.Type() != tt.out.Type() || !DeepEqual(out2, tt.out.Interface()) {
			t.Errorf("ValueOf(%T(%[1]v)).Convert(%s) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t2, out2, tt.out.Interface())
		}

		// vout3 represents a new value of the out type, set to vout2.  This makes
		// sure the converted value vout2 is really usable as a regular value.
		vout3 := New(t2).Elem()
		vout3.Set(vout2)
		out3 := vout3.Interface()
		if vout3.Type() != tt.out.Type() || !DeepEqual(out3, tt.out.Interface()) {
			t.Errorf("Set(ValueOf(%T(%[1]v)).Convert(%s)) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t2, out3, tt.out.Interface())
		}
	}

	// Assume that of all the types we saw during the tests,
	// if there wasn't an explicit entry for a conversion between
	// a pair of types, then it's not to be allowed. This checks for
	// things like 'int64' converting to '*int'.
	for t1 := range all {
		for t2 := range all {
			expectOK := t1 == t2 || canConvert[[2]Type{t1, t2}] || t2.Kind() == Interface && t2.NumMethod() == 0
			if ok := t1.ConvertibleTo(t2); ok != expectOK {
				t.Errorf("(%s).ConvertibleTo(%s) = %v, want %v", t1, t2, ok, expectOK)
			}
		}
	}
}

type ComparableStruct struct {
	X int
}

type NonComparableStruct struct {
	X int
	Y map[string]int
}

var comparableTests = []struct {
	typ Type
	ok  bool
}{
	{typ: TypeOf(1), ok: true},
	{typ: TypeOf("hello"), ok: true},
	{typ: TypeOf(new(byte)), ok: true},
	{typ: TypeOf((func())(nil)), ok: false},
	{typ: TypeOf([]byte{}), ok: false},
	{typ: TypeOf(map[string]int{}), ok: false},
	{typ: TypeOf(make(chan int)), ok: true},
	{typ: TypeOf(1.5), ok: true},
	{typ: TypeOf(false), ok: true},
	{typ: TypeOf(1i), ok: true},
	{typ: TypeOf(ComparableStruct{}), ok: true},
	{typ: TypeOf(NonComparableStruct{}), ok: false},
	{typ: TypeOf([10]map[string]int{}), ok: false},
	{typ: TypeOf([10]string{}), ok: true},
	{typ: TypeOf(new(any)).Elem(), ok: true},
}

func TestComparable(t *testing.T) {
	for _, tt := range comparableTests {
		if ok := tt.typ.Comparable(); ok != tt.ok {
			t.Errorf("TypeOf(%v).Comparable() = %v, want %v", tt.typ, ok, tt.ok)
		}
	}
}

func TestOverflow(t *testing.T) {
	if ovf := V(float64(0)).OverflowFloat(1e300); ovf {
		t.Errorf("%v wrongly overflows float64", 1e300)
	}

	maxFloat32 := float64((1<<24 - 1) << (127 - 23))
	if ovf := V(float32(0)).OverflowFloat(maxFloat32); ovf {
		t.Errorf("%v wrongly overflows float32", maxFloat32)
	}
	ovfFloat32 := float64((1<<24-1)<<(127-23) + 1<<(127-52))
	if ovf := V(float32(0)).OverflowFloat(ovfFloat32); !ovf {
		t.Errorf("%v should overflow float32", ovfFloat32)
	}
	if ovf := V(float32(0)).OverflowFloat(-ovfFloat32); !ovf {
		t.Errorf("%v should overflow float32", -ovfFloat32)
	}

	maxInt32 := int64(0x7fffffff)
	if ovf := V(int32(0)).OverflowInt(maxInt32); ovf {
		t.Errorf("%v wrongly overflows int32", maxInt32)
	}
	if ovf := V(int32(0)).OverflowInt(-1 << 31); ovf {
		t.Errorf("%v wrongly overflows int32", -int64(1)<<31)
	}
	ovfInt32 := int64(1 << 31)
	if ovf := V(int32(0)).OverflowInt(ovfInt32); !ovf {
		t.Errorf("%v should overflow int32", ovfInt32)
	}

	maxUint32 := uint64(0xffffffff)
	if ovf := V(uint32(0)).OverflowUint(maxUint32); ovf {
		t.Errorf("%v wrongly overflows uint32", maxUint32)
	}
	ovfUint32 := uint64(1 << 32)
	if ovf := V(uint32(0)).OverflowUint(ovfUint32); !ovf {
		t.Errorf("%v should overflow uint32", ovfUint32)
	}
}

func checkSameType(t *testing.T, x Type, y any) {
	if x != TypeOf(y) || TypeOf(Zero(x).Interface()) != TypeOf(y) {
		t.Errorf("did not find preexisting type for %s (vs %s)", TypeOf(x), TypeOf(y))
	}
}

func TestArrayOf(t *testing.T) {
	// check construction and use of type not in binary
	tests := []struct {
		n          int
		value      func(i int) any
		comparable bool
		want       string
	}{
		{
			n:          0,
			value:      func(i int) any { type Tint int; return Tint(i) },
			comparable: true,
			want:       "[]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tint int; return Tint(i) },
			comparable: true,
			want:       "[0 1 2 3 4 5 6 7 8 9]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tfloat float64; return Tfloat(i) },
			comparable: true,
			want:       "[0 1 2 3 4 5 6 7 8 9]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tstring string; return Tstring(strconv.Itoa(i)) },
			comparable: true,
			want:       "[0 1 2 3 4 5 6 7 8 9]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tstruct struct{ V int }; return Tstruct{V: i} },
			comparable: true,
			want:       "[{0} {1} {2} {3} {4} {5} {6} {7} {8} {9}]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tint int; return []Tint{Tint(i)} },
			comparable: false,
			want:       "[[0] [1] [2] [3] [4] [5] [6] [7] [8] [9]]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tint int; return [1]Tint{Tint(i)} },
			comparable: true,
			want:       "[[0] [1] [2] [3] [4] [5] [6] [7] [8] [9]]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tstruct struct{ V [1]int }; return Tstruct{V: [1]int{i}} },
			comparable: true,
			want:       "[{[0]} {[1]} {[2]} {[3]} {[4]} {[5]} {[6]} {[7]} {[8]} {[9]}]",
		},
		{
			n:          10,
			value:      func(i int) any { type Tstruct struct{ V []int }; return Tstruct{V: []int{i}} },
			comparable: false,
			want:       "[{[0]} {[1]} {[2]} {[3]} {[4]} {[5]} {[6]} {[7]} {[8]} {[9]}]",
		},
		{
			n:          10,
			value:      func(i int) any { type TstructUV struct{ U, V int }; return TstructUV{U: i, V: i} },
			comparable: true,
			want:       "[{0 0} {1 1} {2 2} {3 3} {4 4} {5 5} {6 6} {7 7} {8 8} {9 9}]",
		},
		{
			n: 10,
			value: func(i int) any {
				type TstructUV struct {
					U int
					V float64
				}
				return TstructUV{U: i, V: float64(i)}
			},
			comparable: true,
			want:       "[{0 0} {1 1} {2 2} {3 3} {4 4} {5 5} {6 6} {7 7} {8 8} {9 9}]",
		},
	}

	for _, table := range tests {
		at := ArrayOf(table.n, TypeOf(table.value(0)))
		v := New(at).Elem()
		vok := New(at).Elem()
		vnot := New(at).Elem()
		for i := 0; i < v.Len(); i++ {
			v.Index(i).Set(ValueOf(table.value(i)))
			vok.Index(i).Set(ValueOf(table.value(i)))
			j := i
			if i+1 == v.Len() {
				j = i + 1
			}
			vnot.Index(i).Set(ValueOf(table.value(j))) // make it differ only by last element
		}
		s := fmt.Sprint(v.Interface())
		if s != table.want {
			t.Errorf("constructed array = %s, want %s", s, table.want)
		}

		if table.comparable != at.Comparable() {
			t.Errorf("constructed array (%#v) is comparable=%v, want=%v", v.Interface(), at.Comparable(), table.comparable)
		}
		if table.comparable {
			if table.n > 0 {
				if DeepEqual(vnot.Interface(), v.Interface()) {
					t.Errorf(
						"arrays (%#v) compare ok (but should not)",
						v.Interface(),
					)
				}
			}
			if !DeepEqual(vok.Interface(), v.Interface()) {
				t.Errorf(
					"arrays (%#v) compare NOT-ok (but should)",
					v.Interface(),
				)
			}
		}
	}

	// check that type already in binary is found
	type T int
	checkSameType(t, ArrayOf(5, TypeOf(T(1))), [5]T{})
}

func TestArrayOfGC(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := New(ArrayOf(n, tt)).Elem()
		for j := 0; j < v.Len(); j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.Index(j).Set(ValueOf(p).Convert(tt))
		}
		x = append(x, v.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi)
		for j := 0; j < v.Len(); j++ {
			k := v.Index(j).Elem().Interface()
			if k != uintptr(i*n+j) {
				t.Errorf("lost x[%d][%d] = %d, want %d", i, j, k, i*n+j)
			}
		}
	}
}

func TestArrayOfAlg(t *testing.T) {
	at := ArrayOf(6, TypeOf(byte(0)))
	v1 := New(at).Elem()
	v2 := New(at).Elem()
	if v1.Interface() != v1.Interface() {
		t.Errorf("constructed array %v not equal to itself", v1.Interface())
	}
	v1.Index(5).Set(ValueOf(byte(1)))
	if i1, i2 := v1.Interface(), v2.Interface(); i1 == i2 {
		t.Errorf("constructed arrays %v and %v should not be equal", i1, i2)
	}

	at = ArrayOf(6, TypeOf([]int(nil)))
	v1 = New(at).Elem()
	shouldPanic(func() { _ = v1.Interface() == v1.Interface() })
}

func TestArrayOfGenericAlg(t *testing.T) {
	at1 := ArrayOf(5, TypeOf(string("")))
	at := ArrayOf(6, at1)
	v1 := New(at).Elem()
	v2 := New(at).Elem()
	if v1.Interface() != v1.Interface() {
		t.Errorf("constructed array %v not equal to itself", v1.Interface())
	}

	v1.Index(0).Index(0).Set(ValueOf("abc"))
	v2.Index(0).Index(0).Set(ValueOf("efg"))
	if i1, i2 := v1.Interface(), v2.Interface(); i1 == i2 {
		t.Errorf("constructed arrays %v and %v should not be equal", i1, i2)
	}

	v1.Index(0).Index(0).Set(ValueOf("abc"))
	v2.Index(0).Index(0).Set(ValueOf((v1.Index(0).Index(0).String() + " ")[:3]))
	if i1, i2 := v1.Interface(), v2.Interface(); i1 != i2 {
		t.Errorf("constructed arrays %v and %v should be equal", i1, i2)
	}

	// Test hash
	m := MakeMap(MapOf(at, TypeOf(int(0))))
	m.SetMapIndex(v1, ValueOf(1))
	if i1, i2 := v1.Interface(), v2.Interface(); !m.MapIndex(v2).IsValid() {
		t.Errorf("constructed arrays %v and %v have different hashes", i1, i2)
	}
}

func TestArrayOfDirectIface(t *testing.T) {
	{
		type T [1]*byte
		i1 := Zero(TypeOf(T{})).Interface()
		v1 := ValueOf(&i1).Elem()
		p1 := v1.InterfaceData()[1]

		i2 := Zero(ArrayOf(1, PtrTo(TypeOf(int8(0))))).Interface()
		v2 := ValueOf(&i2).Elem()
		p2 := v2.InterfaceData()[1]

		if p1 != 0 {
			t.Errorf("got p1=%v. want=%v", p1, nil)
		}

		if p2 != 0 {
			t.Errorf("got p2=%v. want=%v", p2, nil)
		}
	}
	{
		type T [0]*byte
		i1 := Zero(TypeOf(T{})).Interface()
		v1 := ValueOf(&i1).Elem()
		p1 := v1.InterfaceData()[1]

		i2 := Zero(ArrayOf(0, PtrTo(TypeOf(int8(0))))).Interface()
		v2 := ValueOf(&i2).Elem()
		p2 := v2.InterfaceData()[1]

		if p1 == 0 {
			t.Errorf("got p1=%v. want=not-%v", p1, nil)
		}

		if p2 == 0 {
			t.Errorf("got p2=%v. want=not-%v", p2, nil)
		}
	}
}

func TestSliceOf(t *testing.T) {
	// check construction and use of type not in binary
	type T int
	st := SliceOf(TypeOf(T(1)))
	if got, want := st.String(), "[]reflect_test.T"; got != want {
		t.Errorf("SliceOf(T(1)).String()=%q, want %q", got, want)
	}
	v := MakeSlice(st, 10, 10)
	runtime.GC()
	for i := 0; i < v.Len(); i++ {
		v.Index(i).Set(ValueOf(T(i)))
		runtime.GC()
	}
	s := fmt.Sprint(v.Interface())
	want := "[0 1 2 3 4 5 6 7 8 9]"
	if s != want {
		t.Errorf("constructed slice = %s, want %s", s, want)
	}

	// check that type already in binary is found
	type T1 int
	checkSameType(t, SliceOf(TypeOf(T1(1))), []T1{})
}

func TestSliceOverflow(t *testing.T) {
	// check that MakeSlice panics when size of slice overflows uint
	const S = 1e6
	s := uint(S)
	l := (1<<(unsafe.Sizeof((*byte)(nil))*8)-1)/s + 1
	if l*s >= s {
		t.Fatal("slice size does not overflow")
	}
	var x [S]byte
	st := SliceOf(TypeOf(x))
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("slice overflow does not panic")
		}
	}()
	MakeSlice(st, int(l), int(l))
}

func TestSliceOfGC(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	st := SliceOf(tt)
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := MakeSlice(st, n, n)
		for j := 0; j < v.Len(); j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.Index(j).Set(ValueOf(p).Convert(tt))
		}
		x = append(x, v.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi)
		for j := 0; j < v.Len(); j++ {
			k := v.Index(j).Elem().Interface()
			if k != uintptr(i*n+j) {
				t.Errorf("lost x[%d][%d] = %d, want %d", i, j, k, i*n+j)
			}
		}
	}
}

func TestStructOfFieldName(t *testing.T) {
	// invalid field name "1nvalid"
	shouldPanic(func() {
		StructOf([]StructField{
			{Name: "valid", Type: TypeOf("")},
			{Name: "1nvalid", Type: TypeOf("")},
		})
	})

	// invalid field name "+"
	shouldPanic(func() {
		StructOf([]StructField{
			{Name: "val1d", Type: TypeOf("")},
			{Name: "+", Type: TypeOf("")},
		})
	})

	// no field name
	shouldPanic(func() {
		StructOf([]StructField{
			{Name: "", Type: TypeOf("")},
		})
	})

	// verify creation of a struct with valid struct fields
	validFields := []StructField{
		{
			Name: "φ",
			Type: TypeOf(""),
		},
		{
			Name: "ValidName",
			Type: TypeOf(""),
		},
		{
			Name: "Val1dNam5",
			Type: TypeOf(""),
		},
	}

	validStruct := StructOf(validFields)

	const structStr = `struct { φ string; ValidName string; Val1dNam5 string }`
	if got, want := validStruct.String(), structStr; got != want {
		t.Errorf("StructOf(validFields).String()=%q, want %q", got, want)
	}
}

func TestStructOf(t *testing.T) {
	// check construction and use of type not in binary
	fields := []StructField{
		{
			Name: "S",
			Tag:  "s",
			Type: TypeOf(""),
		},
		{
			Name: "X",
			Tag:  "x",
			Type: TypeOf(byte(0)),
		},
		{
			Name: "Y",
			Type: TypeOf(uint64(0)),
		},
		{
			Name: "Z",
			Type: TypeOf([3]uint16{}),
		},
	}

	st := StructOf(fields)
	v := New(st).Elem()
	runtime.GC()
	v.FieldByName("X").Set(ValueOf(byte(2)))
	v.FieldByIndex([]int{1}).Set(ValueOf(byte(1)))
	runtime.GC()

	s := fmt.Sprint(v.Interface())
	want := `{ 1 0 [0 0 0]}`
	if s != want {
		t.Errorf("constructed struct = %s, want %s", s, want)
	}
	const stStr = `struct { S string "s"; X uint8 "x"; Y uint64; Z [3]uint16 }`
	if got, want := st.String(), stStr; got != want {
		t.Errorf("StructOf(fields).String()=%q, want %q", got, want)
	}

	// check the size, alignment and field offsets
	stt := TypeOf(struct {
		String string
		X      byte
		Y      uint64
		Z      [3]uint16
	}{})
	if st.Size() != stt.Size() {
		t.Errorf("constructed struct size = %v, want %v", st.Size(), stt.Size())
	}
	if st.Align() != stt.Align() {
		t.Errorf("constructed struct align = %v, want %v", st.Align(), stt.Align())
	}
	if st.FieldAlign() != stt.FieldAlign() {
		t.Errorf("constructed struct field align = %v, want %v", st.FieldAlign(), stt.FieldAlign())
	}
	for i := 0; i < st.NumField(); i++ {
		o1 := st.Field(i).Offset
		o2 := stt.Field(i).Offset
		if o1 != o2 {
			t.Errorf("constructed struct field %v offset = %v, want %v", i, o1, o2)
		}
	}

	// Check size and alignment with a trailing zero-sized field.
	st = StructOf([]StructField{
		{
			Name: "F1",
			Type: TypeOf(byte(0)),
		},
		{
			Name: "F2",
			Type: TypeOf([0]*byte{}),
		},
	})
	stt = TypeOf(struct {
		G1 byte
		G2 [0]*byte
	}{})
	if st.Size() != stt.Size() {
		t.Errorf("constructed zero-padded struct size = %v, want %v", st.Size(), stt.Size())
	}
	if st.Align() != stt.Align() {
		t.Errorf("constructed zero-padded struct align = %v, want %v", st.Align(), stt.Align())
	}
	if st.FieldAlign() != stt.FieldAlign() {
		t.Errorf("constructed zero-padded struct field align = %v, want %v", st.FieldAlign(), stt.FieldAlign())
	}
	for i := 0; i < st.NumField(); i++ {
		o1 := st.Field(i).Offset
		o2 := stt.Field(i).Offset
		if o1 != o2 {
			t.Errorf("constructed zero-padded struct field %v offset = %v, want %v", i, o1, o2)
		}
	}

	// check duplicate names
	shouldPanic(func() {
		StructOf([]StructField{
			{Name: "string", Type: TypeOf("")},
			{Name: "string", Type: TypeOf("")},
		})
	})
	shouldPanic(func() {
		StructOf([]StructField{
			{Type: TypeOf("")},
			{Name: "string", Type: TypeOf("")},
		})
	})
	shouldPanic(func() {
		StructOf([]StructField{
			{Type: TypeOf("")},
			{Type: TypeOf("")},
		})
	})
	// check that type already in binary is found
	checkSameType(t, StructOf(fields[2:3]), struct{ Y uint64 }{})

	// gccgo used to fail this test.
	type structFieldType any
	checkSameType(t,
		StructOf([]StructField{
			{
				Name: "F",
				Type: TypeOf((*structFieldType)(nil)).Elem(),
			},
		}),
		struct{ F structFieldType }{})
}

// isExported reports whether name is an exported Go symbol
// (that is, whether it begins with an upper-case letter).
func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func TestStructOfGC(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	fields := []StructField{
		{Name: "X", Type: tt},
		{Name: "Y", Type: tt},
	}
	st := StructOf(fields)

	const n = 10000
	var x []any
	for i := 0; i < n; i++ {
		v := New(st).Elem()
		for j := 0; j < v.NumField(); j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.Field(j).Set(ValueOf(p).Convert(tt))
		}
		x = append(x, v.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi)
		for j := 0; j < v.NumField(); j++ {
			k := v.Field(j).Elem().Interface()
			if k != uintptr(i*n+j) {
				t.Errorf("lost x[%d].%c = %d, want %d", i, "XY"[j], k, i*n+j)
			}
		}
	}
}

func TestStructOfAlg(t *testing.T) {
	st := StructOf([]StructField{{Name: "X", Tag: "x", Type: TypeOf(int(0))}})
	v1 := New(st).Elem()
	v2 := New(st).Elem()
	if !DeepEqual(v1.Interface(), v1.Interface()) {
		t.Errorf("constructed struct %v not equal to itself", v1.Interface())
	}
	v1.FieldByName("X").Set(ValueOf(int(1)))
	if i1, i2 := v1.Interface(), v2.Interface(); DeepEqual(i1, i2) {
		t.Errorf("constructed structs %v and %v should not be equal", i1, i2)
	}

	st = StructOf([]StructField{{Name: "X", Tag: "x", Type: TypeOf([]int(nil))}})
	v1 = New(st).Elem()
	shouldPanic(func() { _ = v1.Interface() == v1.Interface() })
}

func TestStructOfGenericAlg(t *testing.T) {
	st1 := StructOf([]StructField{
		{Name: "X", Tag: "x", Type: TypeOf(int64(0))},
		{Name: "Y", Type: TypeOf(string(""))},
	})
	st := StructOf([]StructField{
		{Name: "S0", Type: st1},
		{Name: "S1", Type: st1},
	})

	tests := []struct {
		rt  Type
		idx []int
	}{
		{
			rt:  st,
			idx: []int{0, 1},
		},
		{
			rt:  st1,
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf([0]int{})},
					{Name: "YY", Type: TypeOf("")},
				},
			),
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf([0]int{})},
					{Name: "YY", Type: TypeOf("")},
					{Name: "ZZ", Type: TypeOf([2]int{})},
				},
			),
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf([1]int{})},
					{Name: "YY", Type: TypeOf("")},
				},
			),
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf([1]int{})},
					{Name: "YY", Type: TypeOf("")},
					{Name: "ZZ", Type: TypeOf([1]int{})},
				},
			),
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf([2]int{})},
					{Name: "YY", Type: TypeOf("")},
					{Name: "ZZ", Type: TypeOf([2]int{})},
				},
			),
			idx: []int{1},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf(int64(0))},
					{Name: "YY", Type: TypeOf(byte(0))},
					{Name: "ZZ", Type: TypeOf("")},
				},
			),
			idx: []int{2},
		},
		{
			rt: StructOf(
				[]StructField{
					{Name: "XX", Type: TypeOf(int64(0))},
					{Name: "YY", Type: TypeOf(int64(0))},
					{Name: "ZZ", Type: TypeOf("")},
					{Name: "AA", Type: TypeOf([1]int64{})},
				},
			),
			idx: []int{2},
		},
	}

	for _, table := range tests {
		v1 := New(table.rt).Elem()
		v2 := New(table.rt).Elem()

		if !DeepEqual(v1.Interface(), v1.Interface()) {
			t.Errorf("constructed struct %v not equal to itself", v1.Interface())
		}

		v1.FieldByIndex(table.idx).Set(ValueOf("abc"))
		v2.FieldByIndex(table.idx).Set(ValueOf("def"))
		if i1, i2 := v1.Interface(), v2.Interface(); DeepEqual(i1, i2) {
			t.Errorf("constructed structs %v and %v should not be equal", i1, i2)
		}

		abc := "abc"
		v1.FieldByIndex(table.idx).Set(ValueOf(abc))
		val := "+" + abc + "-"
		v2.FieldByIndex(table.idx).Set(ValueOf(val[1:4]))
		if i1, i2 := v1.Interface(), v2.Interface(); !DeepEqual(i1, i2) {
			t.Errorf("constructed structs %v and %v should be equal", i1, i2)
		}

		// Test hash
		m := MakeMap(MapOf(table.rt, TypeOf(int(0))))
		m.SetMapIndex(v1, ValueOf(1))
		if i1, i2 := v1.Interface(), v2.Interface(); !m.MapIndex(v2).IsValid() {
			t.Errorf("constructed structs %#v and %#v have different hashes", i1, i2)
		}

		v2.FieldByIndex(table.idx).Set(ValueOf("abc"))
		if i1, i2 := v1.Interface(), v2.Interface(); !DeepEqual(i1, i2) {
			t.Errorf("constructed structs %v and %v should be equal", i1, i2)
		}

		if i1, i2 := v1.Interface(), v2.Interface(); !m.MapIndex(v2).IsValid() {
			t.Errorf("constructed structs %v and %v have different hashes", i1, i2)
		}
	}
}

func TestStructOfDirectIface(t *testing.T) {
	{
		type T struct{ X [1]*byte }
		i1 := Zero(TypeOf(T{})).Interface()
		v1 := ValueOf(&i1).Elem()
		p1 := v1.InterfaceData()[1]

		i2 := Zero(StructOf([]StructField{
			{
				Name: "X",
				Type: ArrayOf(1, TypeOf((*int8)(nil))),
			},
		})).Interface()
		v2 := ValueOf(&i2).Elem()
		p2 := v2.InterfaceData()[1]

		if p1 != 0 {
			t.Errorf("got p1=%v. want=%v", p1, nil)
		}

		if p2 != 0 {
			t.Errorf("got p2=%v. want=%v", p2, nil)
		}
	}
	{
		type T struct{ X [0]*byte }
		i1 := Zero(TypeOf(T{})).Interface()
		v1 := ValueOf(&i1).Elem()
		p1 := v1.InterfaceData()[1]

		i2 := Zero(StructOf([]StructField{
			{
				Name: "X",
				Type: ArrayOf(0, TypeOf((*int8)(nil))),
			},
		})).Interface()
		v2 := ValueOf(&i2).Elem()
		p2 := v2.InterfaceData()[1]

		if p1 == 0 {
			t.Errorf("got p1=%v. want=not-%v", p1, nil)
		}

		if p2 == 0 {
			t.Errorf("got p2=%v. want=not-%v", p2, nil)
		}
	}
}

type StructI int

func (i StructI) Get() int { return int(i) }

type StructIPtr int

func (i *StructIPtr) Get() int { return int(*i) }

func (i *StructIPtr) Set(v int) { *(*int)(i) = v }

type SettableStruct struct {
	SettableField int
}

func (p *SettableStruct) Set(v int) { p.SettableField = v }

type SettablePointer struct {
	SettableField *int
}

func (p *SettablePointer) Set(v int) { *p.SettableField = v }

func TestStructOfWithInterface(t *testing.T) {
	const want = 42
	type Iface interface {
		Get() int
	}
	type IfaceSet interface {
		Set(int)
	}
	tests := []struct {
		name string
		typ  Type
		val  Value
		impl bool
	}{
		{
			name: "StructI",
			typ:  TypeOf(StructI(want)),
			val:  ValueOf(StructI(want)),
			impl: true,
		},
		{
			name: "StructI",
			typ:  PtrTo(TypeOf(StructI(want))),
			val: ValueOf(func() any {
				v := StructI(want)
				return &v
			}()),
			impl: true,
		},
		{
			name: "StructIPtr",
			typ:  PtrTo(TypeOf(StructIPtr(want))),
			val: ValueOf(func() any {
				v := StructIPtr(want)
				return &v
			}()),
			impl: true,
		},
		{
			name: "StructIPtr",
			typ:  TypeOf(StructIPtr(want)),
			val:  ValueOf(StructIPtr(want)),
			impl: false,
		},
		// {
		//	typ:  TypeOf((*Iface)(nil)).Elem(), // FIXME(sbinet): fix method.ifn/tfn
		//	val:  ValueOf(StructI(want)),
		//	impl: true,
		// },
	}

	for i, table := range tests {
		for j := 0; j < 2; j++ {
			var fields []StructField
			if j == 1 {
				fields = append(fields, StructField{
					Name:    "Dummy",
					PkgPath: "",
					Type:    TypeOf(int(0)),
				})
			}
			fields = append(fields, StructField{
				Name:      table.name,
				Anonymous: true,
				PkgPath:   "",
				Type:      table.typ,
			})

			// We currently do not correctly implement methods
			// for embedded fields other than the first.
			// Therefore, for now, we expect those methods
			// to not exist.  See issues 15924 and 20824.
			// When those issues are fixed, this test of panic
			// should be removed.
			if j == 1 && table.impl {
				func() {
					defer func() {
						if err := recover(); err == nil {
							t.Errorf("test-%d-%d did not panic", i, j)
						}
					}()
					_ = StructOf(fields)
				}()
				continue
			}

			rt := StructOf(fields)
			rv := New(rt).Elem()
			rv.Field(j).Set(table.val)

			if _, ok := rv.Interface().(Iface); ok != table.impl {
				if table.impl {
					t.Errorf("test-%d-%d: type=%v fails to implement Iface.\n", i, j, table.typ)
				} else {
					t.Errorf("test-%d-%d: type=%v should NOT implement Iface\n", i, j, table.typ)
				}
				continue
			}

			if !table.impl {
				continue
			}

			v := rv.Interface().(Iface).Get()
			if v != want {
				t.Errorf("test-%d-%d: x.Get()=%v. want=%v\n", i, j, v, want)
			}

			fct := rv.MethodByName("Get")
			out := fct.Call(nil)
			if !DeepEqual(out[0].Interface(), want) {
				t.Errorf("test-%d-%d: x.Get()=%v. want=%v\n", i, j, out[0].Interface(), want)
			}
		}
	}

	// Test an embedded nil pointer with pointer methods.
	fields := []StructField{{
		Name:      "StructIPtr",
		Anonymous: true,
		Type:      PtrTo(TypeOf(StructIPtr(want))),
	}}
	rt := StructOf(fields)
	rv := New(rt).Elem()
	// This should panic since the pointer is nil.
	shouldPanic(func() {
		rv.Interface().(IfaceSet).Set(want)
	})

	// Test an embedded nil pointer to a struct with pointer methods.

	fields = []StructField{{
		Name:      "SettableStruct",
		Anonymous: true,
		Type:      PtrTo(TypeOf(SettableStruct{})),
	}}
	rt = StructOf(fields)
	rv = New(rt).Elem()
	// This should panic since the pointer is nil.
	shouldPanic(func() {
		rv.Interface().(IfaceSet).Set(want)
	})

	// The behavior is different if there is a second field,
	// since now an interface value holds a pointer to the struct
	// rather than just holding a copy of the struct.
	fields = []StructField{
		{
			Name:      "SettableStruct",
			Anonymous: true,
			Type:      PtrTo(TypeOf(SettableStruct{})),
		},
		{
			Name:      "EmptyStruct",
			Anonymous: true,
			Type:      StructOf(nil),
		},
	}
	// With the current implementation this is expected to panic.
	// Ideally it should work and we should be able to see a panic
	// if we call the Set method.
	shouldPanic(func() {
		StructOf(fields)
	})

	// Embed a field that can be stored directly in an interface,
	// with a second field.
	fields = []StructField{
		{
			Name:      "SettablePointer",
			Anonymous: true,
			Type:      TypeOf(SettablePointer{}),
		},
		{
			Name:      "EmptyStruct",
			Anonymous: true,
			Type:      StructOf(nil),
		},
	}
	// With the current implementation this is expected to panic.
	// Ideally it should work and we should be able to call the
	// Set and Get methods.
	shouldPanic(func() {
		StructOf(fields)
	})
}

func TestStructOfTooManyFields(t *testing.T) {
	// Bug Fix: #25402 - this should not panic
	tt := StructOf([]StructField{
		{Name: "Time", Type: TypeOf(time.Time{}), Anonymous: true},
	})

	if _, present := tt.MethodByName("After"); !present {
		t.Errorf("Expected method `After` to be found")
	}
}

func TestChanOf(t *testing.T) {
	// check construction and use of type not in binary
	type T string
	ct := ChanOf(BothDir, TypeOf(T("")))
	v := MakeChan(ct, 2)
	runtime.GC()
	v.Send(ValueOf(T("hello")))
	runtime.GC()
	v.Send(ValueOf(T("world")))
	runtime.GC()

	sv1, _ := v.Recv()
	sv2, _ := v.Recv()
	s1 := sv1.String()
	s2 := sv2.String()
	if s1 != "hello" || s2 != "world" {
		t.Errorf("constructed chan: have %q, %q, want %q, %q", s1, s2, "hello", "world")
	}

	// check that type already in binary is found
	type T1 int
	checkSameType(t, ChanOf(BothDir, TypeOf(T1(1))), (chan T1)(nil))
}

func TestChanOfDir(t *testing.T) {
	// check construction and use of type not in binary
	type T string
	crt := ChanOf(RecvDir, TypeOf(T("")))
	cst := ChanOf(SendDir, TypeOf(T("")))

	// check that type already in binary is found
	type T1 int
	checkSameType(t, ChanOf(RecvDir, TypeOf(T1(1))), (<-chan T1)(nil))
	checkSameType(t, ChanOf(SendDir, TypeOf(T1(1))), (chan<- T1)(nil))

	// check String form of ChanDir
	if crt.ChanDir().String() != "<-chan" {
		t.Errorf("chan dir: have %q, want %q", crt.ChanDir().String(), "<-chan")
	}
	if cst.ChanDir().String() != "chan<-" {
		t.Errorf("chan dir: have %q, want %q", cst.ChanDir().String(), "chan<-")
	}
}

func TestChanOfGC(t *testing.T) {
	done := make(chan bool, 1)
	go func() {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			panic("deadlock in TestChanOfGC")
		}
	}()

	defer func() {
		done <- true
	}()

	type T *uintptr
	tt := TypeOf(T(nil))
	ct := ChanOf(BothDir, tt)

	// NOTE: The garbage collector handles allocated channels specially,
	// so we have to save pointers to channels in x; the pointer code will
	// use the gc info in the newly constructed chan type.
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := MakeChan(ct, n)
		for j := 0; j < n; j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.Send(ValueOf(p).Convert(tt))
		}
		pv := New(ct)
		pv.Elem().Set(v)
		x = append(x, pv.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi).Elem()
		for j := 0; j < n; j++ {
			pv, _ := v.Recv()
			k := pv.Elem().Interface()
			if k != uintptr(i*n+j) {
				t.Errorf("lost x[%d][%d] = %d, want %d", i, j, k, i*n+j)
			}
		}
	}
}

func TestMapOf(t *testing.T) {
	// check construction and use of type not in binary
	type K string
	type V float64

	v := MakeMap(MapOf(TypeOf(K("")), TypeOf(V(0))))
	runtime.GC()
	v.SetMapIndex(ValueOf(K("a")), ValueOf(V(1)))
	runtime.GC()

	s := fmt.Sprint(v.Interface())
	want := "map[a:1]"
	if s != want {
		t.Errorf("constructed map = %s, want %s", s, want)
	}

	// check that type already in binary is found
	checkSameType(t, MapOf(TypeOf(V(0)), TypeOf(K(""))), map[V]K(nil))

	// check that invalid key type panics
	shouldPanic(func() { MapOf(TypeOf((func())(nil)), TypeOf(false)) })
}

func TestMapOfGCKeys(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	mt := MapOf(tt, TypeOf(false))

	// NOTE: The garbage collector handles allocated maps specially,
	// so we have to save pointers to maps in x; the pointer code will
	// use the gc info in the newly constructed map type.
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := MakeMap(mt)
		for j := 0; j < n; j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.SetMapIndex(ValueOf(p).Convert(tt), ValueOf(true))
		}
		pv := New(mt)
		pv.Elem().Set(v)
		x = append(x, pv.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi).Elem()
		var out []int
		for _, kv := range v.MapKeys() {
			out = append(out, int(kv.Elem().Interface().(uintptr)))
		}
		sort.Ints(out)
		for j, k := range out {
			if k != i*n+j {
				t.Errorf("lost x[%d][%d] = %d, want %d", i, j, k, i*n+j)
			}
		}
	}
}

func TestMapOfGCValues(t *testing.T) {
	type T *uintptr
	tt := TypeOf(T(nil))
	mt := MapOf(TypeOf(1), tt)

	// NOTE: The garbage collector handles allocated maps specially,
	// so we have to save pointers to maps in x; the pointer code will
	// use the gc info in the newly constructed map type.
	const n = 100
	var x []any
	for i := 0; i < n; i++ {
		v := MakeMap(mt)
		for j := 0; j < n; j++ {
			p := new(uintptr)
			*p = uintptr(i*n + j)
			v.SetMapIndex(ValueOf(j), ValueOf(p).Convert(tt))
		}
		pv := New(mt)
		pv.Elem().Set(v)
		x = append(x, pv.Interface())
	}
	runtime.GC()

	for i, xi := range x {
		v := ValueOf(xi).Elem()
		for j := 0; j < n; j++ {
			k := v.MapIndex(ValueOf(j)).Elem().Interface().(uintptr)
			if k != uintptr(i*n+j) {
				t.Errorf("lost x[%d][%d] = %d, want %d", i, j, k, i*n+j)
			}
		}
	}
}

func TestFuncOf(t *testing.T) {
	// check construction and use of type not in binary
	type K string
	type V float64

	fn := func(args []Value) []Value {
		if len(args) != 1 {
			t.Errorf("args == %v, want exactly one arg", args)
		} else if args[0].Type() != TypeOf(K("")) {
			t.Errorf("args[0] is type %v, want %v", args[0].Type(), TypeOf(K("")))
		} else if args[0].String() != "gopher" {
			t.Errorf("args[0] = %q, want %q", args[0].String(), "gopher")
		}
		return []Value{ValueOf(V(3.14))}
	}
	v := MakeFunc(FuncOf([]Type{TypeOf(K(""))}, []Type{TypeOf(V(0))}, false), fn)

	outs := v.Call([]Value{ValueOf(K("gopher"))})
	if len(outs) != 1 {
		t.Fatalf("v.Call returned %v, want exactly one result", outs)
	} else if outs[0].Type() != TypeOf(V(0)) {
		t.Fatalf("c.Call[0] is type %v, want %v", outs[0].Type(), TypeOf(V(0)))
	}
	f := outs[0].Float()
	if f != 3.14 {
		t.Errorf("constructed func returned %f, want %f", f, 3.14)
	}

	// check that types already in binary are found
	type T1 int
	testCases := []struct {
		in, out  []Type
		variadic bool
		want     any
	}{
		{in: []Type{TypeOf(T1(0))}, want: (func(T1))(nil)},
		{in: []Type{TypeOf(int(0))}, want: (func(int))(nil)},
		{in: []Type{SliceOf(TypeOf(int(0)))}, variadic: true, want: (func(...int))(nil)},
		{in: []Type{TypeOf(int(0))}, out: []Type{TypeOf(false)}, want: (func(int) bool)(nil)},
		{in: []Type{TypeOf(int(0))}, out: []Type{TypeOf(false), TypeOf("")}, want: (func(int) (bool, string))(nil)},
	}
	for _, tt := range testCases {
		checkSameType(t, FuncOf(tt.in, tt.out, tt.variadic), tt.want)
	}

	// check that variadic requires last element be a slice.
	FuncOf([]Type{TypeOf(1), TypeOf(""), SliceOf(TypeOf(false))}, nil, true)
	shouldPanic(func() { FuncOf([]Type{TypeOf(0), TypeOf(""), TypeOf(false)}, nil, true) })
	shouldPanic(func() { FuncOf(nil, nil, true) })
}

type B1 struct {
	X int
	Y int
	Z int
}

func BenchmarkFieldByName1(b *testing.B) {
	t := TypeOf(B1{})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.FieldByName("Z")
		}
	})
}

func BenchmarkFieldByName2(b *testing.B) {
	t := TypeOf(S3{})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.FieldByName("B")
		}
	})
}

type R0 struct {
	*R1
	*R2
	*R3
	*R4
}

type R1 struct {
	*R5
	*R6
	*R7
	*R8
}

type R2 R1

type R3 R1

type R4 R1

type R5 struct {
	*R9
	*R10
	*R11
	*R12
}

type R6 R5

type R7 R5

type R8 R5

type R9 struct {
	*R13
	*R14
	*R15
	*R16
}

type R10 R9

type R11 R9

type R12 R9

type R13 struct {
	*R17
	*R18
	*R19
	*R20
}

type R14 R13

type R15 R13

type R16 R13

type R17 struct {
	*R21
	*R22
	*R23
	*R24
}

type R18 R17

type R19 R17

type R20 R17

type R21 struct {
	X int
}

type R22 R21

type R23 R21

type R24 R21

func TestEmbed(t *testing.T) {
	typ := TypeOf(R0{})
	f, ok := typ.FieldByName("X")
	if ok {
		t.Fatalf(`FieldByName("X") should fail, returned %v`, f.Index)
	}
}

func BenchmarkFieldByName3(b *testing.B) {
	t := TypeOf(R0{})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.FieldByName("X")
		}
	})
}

type S struct {
	i1 int64
	i2 int64
}

func BenchmarkInterfaceBig(b *testing.B) {
	v := ValueOf(S{})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v.Interface()
		}
	})
	b.StopTimer()
}

func TestAllocsInterfaceBig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	v := ValueOf(S{})
	if allocs := testing.AllocsPerRun(100, func() { v.Interface() }); allocs > 0 {
		t.Error("allocs:", allocs)
	}
}

func BenchmarkInterfaceSmall(b *testing.B) {
	v := ValueOf(int64(0))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v.Interface()
		}
	})
}

func TestAllocsInterfaceSmall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	v := ValueOf(int64(0))
	if allocs := testing.AllocsPerRun(100, func() { v.Interface() }); allocs > 0 {
		t.Error("allocs:", allocs)
	}
}

// An exhaustive is a mechanism for writing exhaustive or stochastic tests.
// The basic usage is:
//
//	for x.Next() {
//		... code using x.Maybe() or x.Choice(n) to create test cases ...
//	}
//
// Each iteration of the loop returns a different set of results, until all
// possible result sets have been explored. It is okay for different code paths
// to make different method call sequences on x, but there must be no
// other source of non-determinism in the call sequences.
//
// When faced with a new decision, x chooses randomly. Future explorations
// of that path will choose successive values for the result. Thus, stopping
// the loop after a fixed number of iterations gives somewhat stochastic
// testing.
//
// Example:
//
//	for x.Next() {
//		v := make([]bool, x.Choose(4))
//		for i := range v {
//			v[i] = x.Maybe()
//		}
//		fmt.Println(v)
//	}
//
// prints (in some order):
//
//	[]
//	[false]
//	[true]
//	[false false]
//	[false true]
//	...
//	[true true]
//	[false false false]
//	...
//	[true true true]
//	[false false false false]
//	...
//	[true true true true]
type exhaustive struct {
	r    *rand.Rand
	pos  int
	last []choice
}

type choice struct {
	off int
	n   int
	max int
}

func (x *exhaustive) Next() bool {
	if x.r == nil {
		x.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	x.pos = 0
	if x.last == nil {
		x.last = []choice{}
		return true
	}
	for i := len(x.last) - 1; i >= 0; i-- {
		c := &x.last[i]
		if c.n+1 < c.max {
			c.n++
			x.last = x.last[:i+1]
			return true
		}
	}
	return false
}

func (x *exhaustive) Choose(max int) int {
	if x.pos >= len(x.last) {
		x.last = append(x.last, choice{off: x.r.Intn(max), n: 0, max: max})
	}
	c := &x.last[x.pos]
	x.pos++
	if c.max != max {
		panic("inconsistent use of exhaustive tester")
	}
	return (c.n + c.off) % max
}

func (x *exhaustive) Maybe() bool {
	return x.Choose(2) == 1
}

func GCFunc(args []Value) []Value {
	runtime.GC()
	return []Value{}
}

func TestReflectFuncTraceback(t *testing.T) {
	f := MakeFunc(TypeOf(func() {}), GCFunc)
	f.Call([]Value{})
}

func TestReflectMethodTraceback(t *testing.T) {
	p := Point{x: 3, y: 4}
	m := ValueOf(p).MethodByName("GCMethod")
	i := ValueOf(m.Interface()).Call([]Value{ValueOf(5)})[0].Int()
	if i != 8 {
		t.Errorf("Call returned %d; want 8", i)
	}
}

func TestBigZero(t *testing.T) {
	const size = 1 << 10
	var v [size]byte
	z := Zero(ValueOf(v).Type()).Interface().([size]byte)
	for i := 0; i < size; i++ {
		if z[i] != 0 {
			t.Fatalf("Zero object not all zero, index %d", i)
		}
	}
}

func TestFieldByIndexNil(t *testing.T) {
	type P struct {
		F int
	}
	type T struct {
		*P
	}
	v := ValueOf(T{})

	v.FieldByName("P") // should be fine

	defer func() {
		if err := recover(); err == nil {
			t.Fatalf("no error")
		} else if !strings.Contains(fmt.Sprint(err), "nil pointer to embedded struct") {
			t.Fatalf(`err=%q, wanted error containing "nil pointer to embedded struct"`, err)
		}
	}()
	v.FieldByName("F") // should panic

	t.Fatalf("did not panic")
}

// Given
//	type Outer struct {
//		*Inner
//		...
//	}
// the compiler generates the implementation of (*Outer).M dispatching to the embedded Inner.
// The implementation is logically:
//	func (p *Outer) M() {
//		(p.Inner).M()
//	}
// but since the only change here is the replacement of one pointer receiver with another,
// the actual generated code overwrites the original receiver with the p.Inner pointer and
// then jumps to the M method expecting the *Inner receiver.
//
// During reflect.Value.Call, we create an argument frame and the associated data structures
// to describe it to the garbage collector, populate the frame, call reflect.call to
// run a function call using that frame, and then copy the results back out of the frame.
// The reflect.call function does a memmove of the frame structure onto the
// stack (to set up the inputs), runs the call, and the memmoves the stack back to
// the frame structure (to preserve the outputs).
//
// Originally reflect.call did not distinguish inputs from outputs: both memmoves
// were for the full stack frame. However, in the case where the called function was
// one of these wrappers, the rewritten receiver is almost certainly a different type
// than the original receiver. This is not a problem on the stack, where we use the
// program counter to determine the type information and understand that
// during (*Outer).M the receiver is an *Outer while during (*Inner).M the receiver in the same
// memory word is now an *Inner. But in the statically typed argument frame created
// by reflect, the receiver is always an *Outer. Copying the modified receiver pointer
// off the stack into the frame will store an *Inner there, and then if a garbage collection
// happens to scan that argument frame before it is discarded, it will scan the *Inner
// memory as if it were an *Outer. If the two have different memory layouts, the
// collection will interpret the memory incorrectly.
//
// One such possible incorrect interpretation is to treat two arbitrary memory words
// (Inner.P1 and Inner.P2 below) as an interface (Outer.R below). Because interpreting
// an interface requires dereferencing the itab word, the misinterpretation will try to
// deference Inner.P1, causing a crash during garbage collection.
//
// This came up in a real program in issue 7725.

type Outer struct {
	*Inner
	R io.Reader
}

type Inner struct {
	X  *Outer
	P1 uintptr
	P2 uintptr
}

func (pi *Inner) M() {
	// Clear references to pi so that the only way the
	// garbage collection will find the pointer is in the
	// argument frame, typed as a *Outer.
	pi.X.Inner = nil

	// Set up an interface value that will cause a crash.
	// P1 = 1 is a non-zero, so the interface looks non-nil.
	// P2 = pi ensures that the data word points into the
	// allocated heap; if not the collection skips the interface
	// value as irrelevant, without dereferencing P1.
	pi.P1 = 1
	pi.P2 = uintptr(unsafe.Pointer(pi))
}

func TestMakeFuncStackCopy(t *testing.T) {
	target := func(in []Value) []Value {
		runtime.GC()
		useStack(16)
		return []Value{ValueOf(9)}
	}

	var concrete func(*int, int) int
	fn := MakeFunc(ValueOf(concrete).Type(), target)
	ValueOf(&concrete).Elem().Set(fn)
	x := concrete(nil, 7)
	if x != 9 {
		t.Errorf("have %#q want 9", x)
	}
}

// use about n KB of stack
func useStack(n int) {
	if n == 0 {
		return
	}
	var b [1024]byte // makes frame about 1KB
	useStack(n - 1 + int(b[99]))
}

type Impl struct{}

func (Impl) F() {}

func TestValueString(t *testing.T) {
	rv := ValueOf(Impl{})
	if rv.String() != "<reflect_test.Impl Value>" {
		t.Errorf("ValueOf(Impl{}).String() = %q, want %q", rv.String(), "<reflect_test.Impl Value>")
	}

	method := rv.Method(0)
	if method.String() != "<func() Value>" {
		t.Errorf("ValueOf(Impl{}).Method(0).String() = %q, want %q", method.String(), "<func() Value>")
	}
}

func TestInvalid(t *testing.T) {
	// Used to have inconsistency between IsValid() and Kind() != Invalid.
	type T struct{ v any }

	v := ValueOf(T{}).Field(0)
	if !v.IsValid() || v.Kind() != Interface {
		t.Errorf("field: IsValid=%v, Kind=%v, want true, Interface", v.IsValid(), v.Kind())
	}
	v = v.Elem()
	if v.IsValid() || v.Kind() != Invalid {
		t.Errorf("field elem: IsValid=%v, Kind=%v, want false, Invalid", v.IsValid(), v.Kind())
	}
}

// Issue 8917.
func TestLargeGCProg(t *testing.T) {
	fv := ValueOf(func([256]*byte) {})
	fv.Call([]Value{ValueOf([256]*byte{})})
}

func fieldIndexRecover(t Type, i int) (recovered any) {
	defer func() {
		recovered = recover()
	}()

	t.Field(i)
	return
}

// Issue 15046.
func TestTypeFieldOutOfRangePanic(t *testing.T) {
	typ := TypeOf(struct{ X int }{X: 10})
	testIndices := [...]struct {
		i         int
		mustPanic bool
	}{
		0: {i: -2, mustPanic: true},
		1: {i: 0, mustPanic: false},
		2: {i: 1, mustPanic: true},
		3: {i: 1 << 10, mustPanic: true},
	}
	for i, tt := range testIndices {
		recoveredErr := fieldIndexRecover(typ, tt.i)
		if tt.mustPanic {
			if recoveredErr == nil {
				t.Errorf("#%d: fieldIndex %d expected to panic", i, tt.i)
			}
		} else {
			if recoveredErr != nil {
				t.Errorf("#%d: got err=%v, expected no panic", i, recoveredErr)
			}
		}
	}
}

// Issue 9179.
func TestCallGC(t *testing.T) {
	f := func(a, b, c, d, e string) {}

	g := func(in []Value) []Value {
		runtime.GC()
		return nil
	}
	typ := ValueOf(f).Type()
	f2 := MakeFunc(typ, g).Interface().(func(string, string, string, string, string))
	f2("four", "five5", "six666", "seven77", "eight888")
}

// Issue 18635 (function version).
func TestKeepFuncLive(t *testing.T) {
	// Test that we keep makeFuncImpl live as long as it is
	// referenced on the stack.
	typ := TypeOf(func(i int) {})
	var f, g func(in []Value) []Value
	f = func(in []Value) []Value {
		clobber()
		i := int(in[0].Int())
		if i > 0 {
			// We can't use Value.Call here because
			// runtime.call* will keep the makeFuncImpl
			// alive. However, by converting it to an
			// interface value and calling that,
			// reflect.callReflect is the only thing that
			// can keep the makeFuncImpl live.
			//
			// Alternate between f and g so that if we do
			// reuse the memory prematurely it's more
			// likely to get obviously corrupted.
			MakeFunc(typ, g).Interface().(func(i int))(i - 1)
		}
		return nil
	}
	g = func(in []Value) []Value {
		clobber()
		i := int(in[0].Int())
		MakeFunc(typ, f).Interface().(func(i int))(i)
		return nil
	}
	MakeFunc(typ, f).Call([]Value{ValueOf(10)})
}

type UnExportedFirst int

func (i UnExportedFirst) ΦExported() {}

func (i UnExportedFirst) unexported() {}

// Issue 21177
func TestMethodByNameUnExportedFirst(t *testing.T) {
	defer func() {
		if recover() != nil {
			t.Errorf("should not panic")
		}
	}()
	typ := TypeOf(UnExportedFirst(0))
	m, _ := typ.MethodByName("ΦExported")
	if m.Name != "ΦExported" {
		t.Errorf("got %s, expected ΦExported", m.Name)
	}
}

// Issue 18635 (method version).
type KeepMethodLive struct{}

func (k KeepMethodLive) Method1(i int) {
	clobber()
	if i > 0 {
		ValueOf(k).MethodByName("Method2").Interface().(func(i int))(i - 1)
	}
}

func (k KeepMethodLive) Method2(i int) {
	clobber()
	ValueOf(k).MethodByName("Method1").Interface().(func(i int))(i)
}

func TestKeepMethodLive(t *testing.T) {
	// Test that we keep methodValue live as long as it is
	// referenced on the stack.
	KeepMethodLive{}.Method1(10)
}

// clobber tries to clobber unreachable memory.
func clobber() {
	runtime.GC()
	for i := 1; i < 32; i++ {
		for j := 0; j < 10; j++ {
			obj := make([]*byte, i)
			sink = obj
		}
	}
	runtime.GC()
}

type funcLayoutTest struct {
	rcvr, t                  Type
	size, argsize, retOffset uintptr
	stack                    []byte // pointer bitmap: 1 is pointer, 0 is scalar
	gc                       []byte
}

var funcLayoutTests []funcLayoutTest

func init() {
	var argAlign uintptr = PtrSize
	if runtime.GOARCH == "amd64p32" {
		argAlign = 2 * PtrSize
	}
	roundup := func(x uintptr, a uintptr) uintptr {
		return (x + a - 1) / a * a
	}

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func(a, b string) string { return "" }).Type(),
			size:      6 * PtrSize,
			argsize:   4 * PtrSize,
			retOffset: 4 * PtrSize,
			stack:     []byte{1, 0, 1, 0, 1},
			gc:        []byte{1, 0, 1, 0, 1},
		})

	var r []byte
	if PtrSize == 4 {
		r = []byte{0, 0, 0, 1}
	} else {
		r = []byte{0, 0, 1}
	}
	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func(a, b, c uint32, p *byte, d uint16) {}).Type(),
			size:      roundup(roundup(3*4, PtrSize)+PtrSize+2, argAlign),
			argsize:   roundup(3*4, PtrSize) + PtrSize + 2,
			retOffset: roundup(roundup(3*4, PtrSize)+PtrSize+2, argAlign),
			stack:     r,
			gc:        r,
		})

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func(a map[int]int, b uintptr, c any) {}).Type(),
			size:      4 * PtrSize,
			argsize:   4 * PtrSize,
			retOffset: 4 * PtrSize,
			stack:     []byte{1, 0, 1, 1},
			gc:        []byte{1, 0, 1, 1},
		})

	type S struct {
		a, b uintptr
		c, d *byte
	}
	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func(a S) {}).Type(),
			size:      4 * PtrSize,
			argsize:   4 * PtrSize,
			retOffset: 4 * PtrSize,
			stack:     []byte{0, 0, 1, 1},
			gc:        []byte{0, 0, 1, 1},
		})

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      ValueOf((*byte)(nil)).Type(),
			t:         ValueOf(func(a uintptr, b *int) {}).Type(),
			size:      roundup(3*PtrSize, argAlign),
			argsize:   3 * PtrSize,
			retOffset: roundup(3*PtrSize, argAlign),
			stack:     []byte{1, 0, 1},
			gc:        []byte{1, 0, 1},
		})

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func(a uintptr) {}).Type(),
			size:      roundup(PtrSize, argAlign),
			argsize:   PtrSize,
			retOffset: roundup(PtrSize, argAlign),
			stack:     []byte{},
			gc:        []byte{},
		})

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      nil,
			t:         ValueOf(func() uintptr { return 0 }).Type(),
			size:      PtrSize,
			argsize:   0,
			retOffset: 0,
			stack:     []byte{},
			gc:        []byte{},
		})

	funcLayoutTests = append(funcLayoutTests,
		funcLayoutTest{
			rcvr:      ValueOf(uintptr(0)).Type(),
			t:         ValueOf(func(a uintptr) {}).Type(),
			size:      2 * PtrSize,
			argsize:   2 * PtrSize,
			retOffset: 2 * PtrSize,
			stack:     []byte{1},
			gc:        []byte{1},
			// Note: this one is tricky, as the receiver is not a pointer. But we
			// pass the receiver by reference to the autogenerated pointer-receiver
			// version of the function.
		})
}

func naclpad() []byte {
	if runtime.GOARCH == "amd64p32" {
		return lit(0)
	}
	return nil
}

func rep(n int, b []byte) []byte { return bytes.Repeat(b, n) }

func join(b ...[]byte) []byte { return bytes.Join(b, nil) }

func lit(x ...byte) []byte { return x }

func TestTypeOfTypeOf(t *testing.T) {
	// Check that all the type constructors return concrete *rtype implementations.
	// It's difficult to test directly because the reflect package is only at arm's length.
	// The easiest thing to do is just call a function that crashes if it doesn't get an *rtype.
	check := func(name string, typ Type) {
		if underlying := TypeOf(typ).String(); underlying != "*reflect.rtype" {
			t.Errorf("%v returned %v, not *reflect.rtype", name, underlying)
		}
	}

	type T struct{ int }
	check("TypeOf", TypeOf(T{}))

	check("ArrayOf", ArrayOf(10, TypeOf(T{})))
	check("ChanOf", ChanOf(BothDir, TypeOf(T{})))
	check("FuncOf", FuncOf([]Type{TypeOf(T{})}, nil, false))
	check("MapOf", MapOf(TypeOf(T{}), TypeOf(T{})))
	check("PtrTo", PtrTo(TypeOf(T{})))
	check("SliceOf", SliceOf(TypeOf(T{})))
}

type XM struct{ _ bool }

func (*XM) String() string { return "" }

func TestPtrToMethods(t *testing.T) {
	var y struct{ XM }
	yp := New(TypeOf(y)).Interface()
	_, ok := yp.(fmt.Stringer)
	if !ok {
		t.Fatal("does not implement Stringer, but should")
	}
}

func TestMapAlloc(t *testing.T) {
	m := ValueOf(make(map[int]int, 10))
	k := ValueOf(5)
	v := ValueOf(7)
	allocs := testing.AllocsPerRun(100, func() {
		m.SetMapIndex(k, v)
	})
	if allocs > 0.5 {
		t.Errorf("allocs per map assignment: want 0 got %f", allocs)
	}

	const size = 1000
	tmp := 0
	val := ValueOf(&tmp).Elem()
	allocs = testing.AllocsPerRun(100, func() {
		mv := MakeMapWithSize(TypeOf(map[int]int{}), size)
		// Only adding half of the capacity to not trigger re-allocations due too many overloaded buckets.
		for i := 0; i < size/2; i++ {
			val.SetInt(int64(i))
			mv.SetMapIndex(val, val)
		}
	})
	if allocs > 10 {
		t.Errorf("allocs per map assignment: want at most 10 got %f", allocs)
	}
	// Empirical testing shows that with capacity hint single run will trigger 3 allocations and without 91. I set
	// the threshold to 10, to not make it overly brittle if something changes in the initial allocation of the
	// map, but to still catch a regression where we keep re-allocating in the hashmap as new entries are added.
}

func TestChanAlloc(t *testing.T) {
	// Note: for a chan int, the return Value must be allocated, so we
	// use a chan *int instead.
	c := ValueOf(make(chan *int, 1))
	v := ValueOf(new(int))
	allocs := testing.AllocsPerRun(100, func() {
		c.Send(v)
		_, _ = c.Recv()
	})
	if allocs < 0.5 || allocs > 1.5 {
		t.Errorf("allocs per chan send/recv: want 1 got %f", allocs)
	}
	// Note: there is one allocation in reflect.recv which seems to be
	// a limitation of escape analysis. If that is ever fixed the
	// allocs < 0.5 condition will trigger and this test should be fixed.
}

type TheNameOfThisTypeIsExactly255BytesLongSoWhenTheCompilerPrependsTheReflectTestPackageNameAndExtraStarTheLinkerRuntimeAndReflectPackagesWillHaveToCorrectlyDecodeTheSecondLengthByte0123456789_0123456789_0123456789_0123456789_0123456789_012345678 int

type nameTest struct {
	v    any
	want string
}

var nameTests = []nameTest{
	{v: (*int32)(nil), want: "int32"},
	{v: (*D1)(nil), want: "D1"},
	{v: (*[]D1)(nil), want: ""},
	{v: (*chan D1)(nil), want: ""},
	{v: (*func() D1)(nil), want: ""},
	{v: (*<-chan D1)(nil), want: ""},
	{v: (*chan<- D1)(nil), want: ""},
	{v: (*any)(nil), want: ""},
	{v: (*interface {
		F()
	})(nil), want: ""},
	{v: (*TheNameOfThisTypeIsExactly255BytesLongSoWhenTheCompilerPrependsTheReflectTestPackageNameAndExtraStarTheLinkerRuntimeAndReflectPackagesWillHaveToCorrectlyDecodeTheSecondLengthByte0123456789_0123456789_0123456789_0123456789_0123456789_012345678)(nil), want: "TheNameOfThisTypeIsExactly255BytesLongSoWhenTheCompilerPrependsTheReflectTestPackageNameAndExtraStarTheLinkerRuntimeAndReflectPackagesWillHaveToCorrectlyDecodeTheSecondLengthByte0123456789_0123456789_0123456789_0123456789_0123456789_012345678"},
}

func TestNames(t *testing.T) {
	for _, test := range nameTests {
		typ := TypeOf(test.v).Elem()
		if got := typ.Name(); got != test.want {
			t.Errorf("%v Name()=%q, want %q", typ, got, test.want)
		}
	}
}

func TestTypeStrings(t *testing.T) {
	type stringTest struct {
		typ  Type
		want string
	}
	stringTests := []stringTest{
		{typ: TypeOf(func(int) {}), want: "func(int)"},
		{typ: FuncOf([]Type{TypeOf(int(0))}, nil, false), want: "func(int)"},
		{typ: TypeOf(XM{}), want: "reflect_test.XM"},
		{typ: TypeOf(new(XM)), want: "*reflect_test.XM"},
		{typ: TypeOf(new(XM).String), want: "func() string"},
		{typ: TypeOf(new(XM)).Method(0).Type, want: "func(*reflect_test.XM) string"},
		{typ: ChanOf(3, TypeOf(XM{})), want: "chan reflect_test.XM"},
		{typ: MapOf(TypeOf(int(0)), TypeOf(XM{})), want: "map[int]reflect_test.XM"},
		{typ: ArrayOf(3, TypeOf(XM{})), want: "[3]reflect_test.XM"},
		{typ: ArrayOf(3, TypeOf(struct{}{})), want: "[3]struct {}"},
	}

	for i, test := range stringTests {
		if got, want := test.typ.String(), test.want; got != want {
			t.Errorf("type %d String()=%q, want %q", i, got, want)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	v := TypeOf(XM{})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			New(v)
		}
	})
}

func TestSwapper(t *testing.T) {
	type I int
	var a, b, c I
	type pair struct {
		x, y int
	}
	type pairPtr struct {
		x, y int
		p    *I
	}
	type S string

	tests := []struct {
		in   any
		i, j int
		want any
	}{
		{
			in:   []int{1, 20, 300},
			i:    0,
			j:    2,
			want: []int{300, 20, 1},
		},
		{
			in:   []uintptr{1, 20, 300},
			i:    0,
			j:    2,
			want: []uintptr{300, 20, 1},
		},
		{
			in:   []int16{1, 20, 300},
			i:    0,
			j:    2,
			want: []int16{300, 20, 1},
		},
		{
			in:   []int8{1, 20, 100},
			i:    0,
			j:    2,
			want: []int8{100, 20, 1},
		},
		{
			in:   []*I{&a, &b, &c},
			i:    0,
			j:    2,
			want: []*I{&c, &b, &a},
		},
		{
			in:   []string{"eric", "sergey", "larry"},
			i:    0,
			j:    2,
			want: []string{"larry", "sergey", "eric"},
		},
		{
			in:   []S{"eric", "sergey", "larry"},
			i:    0,
			j:    2,
			want: []S{"larry", "sergey", "eric"},
		},
		{
			in:   []pair{{x: 1, y: 2}, {x: 3, y: 4}, {x: 5, y: 6}},
			i:    0,
			j:    2,
			want: []pair{{x: 5, y: 6}, {x: 3, y: 4}, {x: 1, y: 2}},
		},
		{
			in:   []pairPtr{{x: 1, y: 2, p: &a}, {x: 3, y: 4, p: &b}, {x: 5, y: 6, p: &c}},
			i:    0,
			j:    2,
			want: []pairPtr{{x: 5, y: 6, p: &c}, {x: 3, y: 4, p: &b}, {x: 1, y: 2, p: &a}},
		},
	}

	for i, tt := range tests {
		inStr := fmt.Sprint(tt.in)
		Swapper(tt.in)(tt.i, tt.j)
		if !DeepEqual(tt.in, tt.want) {
			t.Errorf("%d. swapping %v and %v of %v = %v; want %v", i, tt.i, tt.j, inStr, tt.in, tt.want)
		}
	}
}

// TestUnaddressableField tests that the reflect package will not allow
// a type from another package to be used as a named type with an
// unexported field.
//
// This ensures that unexported fields cannot be modified by other packages.
func TestUnaddressableField(t *testing.T) {
	var b Buffer // type defined in reflect, a different package
	var localBuffer struct {
		buf []byte
	}
	lv := ValueOf(&localBuffer).Elem()
	rv := ValueOf(b)
	shouldPanic(func() {
		lv.Set(rv)
	})
}

type Tint int

type Tint2 = Tint

type Talias1 struct {
	byte
	uint8
	int
	int32
	rune
}

type Talias2 struct {
	Tint
	Tint2
}

func TestAliasNames(t *testing.T) {
	t1 := Talias1{byte: 1, uint8: 2, int: 3, int32: 4, rune: 5}
	out := fmt.Sprintf("%#v", t1)
	want := "reflect_test.Talias1{byte:0x1, uint8:0x2, int:3, int32:4, rune:5}"
	if out != want {
		t.Errorf("Talias1 print:\nhave: %s\nwant: %s", out, want)
	}

	t2 := Talias2{Tint: 1, Tint2: 2}
	out = fmt.Sprintf("%#v", t2)
	want = "reflect_test.Talias2{Tint:1, Tint2:2}"
	if out != want {
		t.Errorf("Talias2 print:\nhave: %s\nwant: %s", out, want)
	}
}

func TestIssue22031(t *testing.T) {
	type s []struct{ C int }

	type t1 struct{ s }
	type t2 struct{ f s }

	tests := []Value{
		ValueOf(t1{s: s{{}}}).Field(0).Index(0).Field(0),
		ValueOf(t2{f: s{{}}}).Field(0).Index(0).Field(0),
	}

	for i, test := range tests {
		if test.CanSet() {
			t.Errorf("%d: CanSet: got true, want false", i)
		}
	}
}

type NonExportedFirst int

func (i NonExportedFirst) ΦExported() {}

func (i NonExportedFirst) nonexported() int { panic("wrong") }

func TestIssue22073(t *testing.T) {
	m := ValueOf(NonExportedFirst(0)).Method(0)

	if got := m.Type().NumOut(); got != 0 {
		t.Errorf("NumOut: got %v, want 0", got)
	}

	// Shouldn't panic.
	m.Call(nil)
}

func TestMapIterNonEmptyMap(t *testing.T) {
	m := map[string]int{"one": 1, "two": 2, "three": 3}
	iter := ValueOf(m).MapRange()
	if got, want := iterateToString(iter), `[one: 1, three: 3, two: 2]`; got != want {
		t.Errorf("iterator returned %s (after sorting), want %s", got, want)
	}
}

func TestMapIterNilMap(t *testing.T) {
	var m map[string]int
	iter := ValueOf(m).MapRange()
	if got, want := iterateToString(iter), `[]`; got != want {
		t.Errorf("non-empty result iteratoring nil map: %s", got)
	}
}

func TestMapIterSafety(t *testing.T) {
	// Using a zero MapIter causes a panic, but not a crash.
	func() {
		defer func() { recover() }()
		new(MapIter).Key()
		t.Fatal("Key did not panic")
	}()
	func() {
		defer func() { recover() }()
		new(MapIter).Value()
		t.Fatal("Value did not panic")
	}()
	func() {
		defer func() { recover() }()
		new(MapIter).Next()
		t.Fatal("Next did not panic")
	}()

	// Calling Key/Value on a MapIter before Next
	// causes a panic, but not a crash.
	var m map[string]int
	iter := ValueOf(m).MapRange()

	func() {
		defer func() { recover() }()
		iter.Key()
		t.Fatal("Key did not panic")
	}()
	func() {
		defer func() { recover() }()
		iter.Value()
		t.Fatal("Value did not panic")
	}()

	// Calling Next, Key, or Value on an exhausted iterator
	// causes a panic, but not a crash.
	iter.Next() // -> false
	func() {
		defer func() { recover() }()
		iter.Key()
		t.Fatal("Key did not panic")
	}()
	func() {
		defer func() { recover() }()
		iter.Value()
		t.Fatal("Value did not panic")
	}()
	func() {
		defer func() { recover() }()
		iter.Next()
		t.Fatal("Next did not panic")
	}()
}

func TestMapIterNext(t *testing.T) {
	// The first call to Next should reflect any
	// insertions to the map since the iterator was created.
	m := map[string]int{}
	iter := ValueOf(m).MapRange()
	m["one"] = 1
	if got, want := iterateToString(iter), `[one: 1]`; got != want {
		t.Errorf("iterator returned deleted elements: got %s, want %s", got, want)
	}
}

func TestMapIterDelete0(t *testing.T) {
	// Delete all elements before first iteration.
	m := map[string]int{"one": 1, "two": 2, "three": 3}
	iter := ValueOf(m).MapRange()
	delete(m, "one")
	delete(m, "two")
	delete(m, "three")
	if got, want := iterateToString(iter), `[]`; got != want {
		t.Errorf("iterator returned deleted elements: got %s, want %s", got, want)
	}
}

func TestMapIterDelete1(t *testing.T) {
	// Delete all elements after first iteration.
	m := map[string]int{"one": 1, "two": 2, "three": 3}
	iter := ValueOf(m).MapRange()
	var got []string
	for iter.Next() {
		got = append(got, fmt.Sprint(iter.Key(), iter.Value()))
		delete(m, "one")
		delete(m, "two")
		delete(m, "three")
	}
	if len(got) != 1 {
		t.Errorf("iterator returned wrong number of elements: got %d, want 1", len(got))
	}
}

// iterateToString returns the set of elements
// returned by an iterator in readable form.
func iterateToString(it *MapIter) string {
	var got []string
	for it.Next() {
		line := fmt.Sprintf("%v: %v", it.Key(), it.Value())
		got = append(got, line)
	}
	sort.Strings(got)
	return "[" + strings.Join(got, ", ") + "]"
}
