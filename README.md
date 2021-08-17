# go-phper-json

PHP flavored encoding/json package

[![Go Reference](https://pkg.go.dev/badge/github.com/shogo82148/go-phper-json.svg)](https://pkg.go.dev/github.com/shogo82148/go-phper-json)
[![Build Status](https://github.com/shogo82148/go-phper-json/workflows/Go/badge.svg)](https://github.com/shogo82148/go-phper-json/actions)

phperjson package works in the same way as the encoding/json package,
but it is useful for dealing with PHP-encoded JSON.
http://php.net/manual/en/function.json-encode.php

Unlike `json.Unmarshal`, `phperjson.Unmarshal` can unmashal a JSON object into a slice.
The key of the object is interpreted as an index of the slice.
It is use for decoding PHP-encoded JSON with JSON_FORCE_OBJECT option.

```go
func ExampleUnmarshal() {
	var jsonBlob = []byte(`[
		{"Name": "Platypus", "Order": "Monotremata"},
		{"Name": "Quoll",    "Order": "Dasyuromorphia"}
	]`)
	type Animal struct {
		Name  string
		Order string
	}

	// phperjson.Unmarshal is compatible with json.Unmarshal.
	var animals1 []Animal
	if err := phperjson.Unmarshal(jsonBlob, &animals1); err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%+v\n", animals1)

	// JSON encoded by PHP with JSON_FORCE_OBJECT option.
	var phpJSONBlob = []byte(`{
		"0": {"Name": "Platypus", "Order": "Monotremata"},
		"1": {"Name": "Quoll",    "Order": "Dasyuromorphia"}
	}`)
	var animals2 []Animal
	if err := phperjson.Unmarshal(phpJSONBlob, &animals2); err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%+v\n", animals2)

	// jsonBlob and phperJSONBlob are equal for PHP
	fmt.Println(reflect.DeepEqual(animals1, animals2))

	// Output:
	// [{Name:Platypus Order:Monotremata} {Name:Quoll Order:Dasyuromorphia}]
	// [{Name:Platypus Order:Monotremata} {Name:Quoll Order:Dasyuromorphia}]
	// true
}
```

And more, you can use ``Type Juggling'' of PHP.
For example, phperjson.Unmarshal can unmashal a JSON string into int,
if the string can be parsed as number.
See http://php.net/manual/en/language.types.type-juggling.php for more detail.

```go
func ExampleUnmarshal_typeJaggling() {
	var jsonBlob = []byte(`{
		"R": 98,
		"G": "218",
		"B": 255.0
	}`)
	type RGB struct {
		R uint8
		G uint8
		B uint8
	}
	// phperjson.Unmarshal is compatible with json.Unmarshal.
	var color RGB
	if err := phperjson.Unmarshal(jsonBlob, &color); err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%+v\n", color)

	// Output:
	// {R:98 G:218 B:255}
}
```

## Benchmark

```
$ go version
go version go1.17 darwin/amd64
$ go test -bench . -benchmem
goos: darwin
goarch: amd64
pkg: github.com/shogo82148/go-phper-json
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
BenchmarkUnicodeDecoder/json-8           5667753               212.0 ns/op        66.04 MB/s          28 B/op          2 allocs/op
BenchmarkUnicodeDecoder/phper-json-8     3428779               349.4 ns/op        40.07 MB/s          60 B/op          4 allocs/op
BenchmarkCodeUnmarshal/json-8                141           8093536 ns/op         239.76 MB/s     3045352 B/op      92670 allocs/op
BenchmarkCodeUnmarshal/phper-json-8           70          16798590 ns/op         115.51 MB/s    16109564 B/op     566553 allocs/op
BenchmarkUnmarshalString/json-8         13276536                87.99 ns/op          160 B/op          2 allocs/op
BenchmarkUnmarshalString/phper-json-8    2155824               538.7 ns/op          2464 B/op          7 allocs/op
BenchmarkUnmarshalFloat64/json-8        14083710                87.76 ns/op          148 B/op          2 allocs/op
BenchmarkUnmarshalFloat64/phper-json-8   1970341               543.4 ns/op          2452 B/op          7 allocs/op
BenchmarkUnmarshalInt64/json-8          17226661                69.99 ns/op          144 B/op          1 allocs/op
BenchmarkUnmarshalInt64/phper-json-8     1932265               620.2 ns/op          2448 B/op          6 allocs/op
BenchmarkUnmapped/json-8                 4018314               290.5 ns/op           200 B/op          4 allocs/op
BenchmarkUnmapped/phper-json-8            963920              1194 ns/op            2256 B/op         28 allocs/op
```
