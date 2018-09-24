// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2018 Shogo Ichinose. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Large data benchmark.
// The JSON data is a summary of agl's changes in the
// go, webkit, and chromium open source projects.
// We benchmark converting between the JSON form
// and in-memory data structures.

package phperjson

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

type codeResponse struct {
	Tree     *codeNode `json:"tree"`
	Username string    `json:"username"`
}

type codeNode struct {
	Name     string      `json:"name"`
	Kids     []*codeNode `json:"kids"`
	CLWeight float64     `json:"cl_weight"`
	Touches  int         `json:"touches"`
	MinT     int64       `json:"min_t"`
	MaxT     int64       `json:"max_t"`
	MeanT    int64       `json:"mean_t"`
}

var codeJSON []byte
var codeStruct codeResponse

func codeInit() {
	f, err := os.Open("testdata/code.json.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	codeJSON = data

	if err := Unmarshal(codeJSON, &codeStruct); err != nil {
		panic("unmarshal code.json: " + err.Error())
	}

	if data, err = Marshal(&codeStruct); err != nil {
		panic("marshal code.json: " + err.Error())
	}

	if !bytes.Equal(data, codeJSON) {
		println("different lengths", len(data), len(codeJSON))
		for i := 0; i < len(data) && i < len(codeJSON); i++ {
			if data[i] != codeJSON[i] {
				println("re-marshal: changed at byte", i)
				println("orig: ", string(codeJSON[i-10:i+10]))
				println("new: ", string(data[i-10:i+10]))
				break
			}
		}
		panic("re-marshal code.json: different result")
	}
}

func BenchmarkUnicodeDecoder(b *testing.B) {
	j := []byte(`"\uD83D\uDE01"`)
	b.Run("json", func(b *testing.B) {
		b.SetBytes(int64(len(j)))
		r := bytes.NewReader(j)
		dec := json.NewDecoder(r)
		var out string
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := dec.Decode(&out); err != nil {
				b.Fatal("Decode:", err)
			}
			r.Seek(0, 0)
		}
	})
	b.Run("phper-json", func(b *testing.B) {
		b.SetBytes(int64(len(j)))
		r := bytes.NewReader(j)
		dec := NewDecoder(r)
		var out string
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := dec.Decode(&out); err != nil {
				b.Fatal("Decode:", err)
			}
			r.Seek(0, 0)
		}
	})
}

func BenchmarkCodeUnmarshal(b *testing.B) {
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}
	b.Run("json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var r codeResponse
				if err := json.Unmarshal(codeJSON, &r); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
		b.SetBytes(int64(len(codeJSON)))
	})
	b.Run("phper-json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var r codeResponse
				if err := Unmarshal(codeJSON, &r); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
		b.SetBytes(int64(len(codeJSON)))
	})
}

func BenchmarkUnmarshalString(b *testing.B) {
	data := []byte(`"hello, world"`)
	b.Run("json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var s string
			for pb.Next() {
				if err := json.Unmarshal(data, &s); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
	b.Run("phper-json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var s string
			for pb.Next() {
				if err := Unmarshal(data, &s); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
}

func BenchmarkUnmarshalFloat64(b *testing.B) {
	data := []byte(`3.14`)
	b.Run("json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var f float64
			for pb.Next() {
				if err := json.Unmarshal(data, &f); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
	b.Run("phper-json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var f float64
			for pb.Next() {
				if err := Unmarshal(data, &f); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
}

func BenchmarkUnmarshalInt64(b *testing.B) {
	data := []byte(`3`)
	b.Run("json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var x int64
			for pb.Next() {
				if err := json.Unmarshal(data, &x); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
	b.Run("phper-json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var x int64
			for pb.Next() {
				if err := Unmarshal(data, &x); err != nil {
					b.Fatal("Unmarshal:", err)
				}
			}
		})
	})
}

func BenchmarkUnmapped(b *testing.B) {
	b.ReportAllocs()
	j := []byte(`{"s": "hello", "y": 2, "o": {"x": 0}, "a": [1, 99, {"x": 1}]}`)
	b.Run("json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var s struct{}
			for pb.Next() {
				if err := json.Unmarshal(j, &s); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
	b.Run("phper-json", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var s struct{}
			for pb.Next() {
				if err := Unmarshal(j, &s); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
