# go-phper-json
PHP flavored encoding/json package

[![GoDoc](https://godoc.org/github.com/shogo82148/go-phper-json?status.svg)](https://godoc.org/github.com/shogo82148/go-phper-json)

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
$ go test -bench . -benchmem
goos: darwin
goarch: amd64
pkg: github.com/shogo82148/go-phper-json
BenchmarkUnicodeDecoder/json-4           5000000               274 ns/op          51.04 MB/s          36 B/op          2 allocs/op
BenchmarkUnicodeDecoder/phper-json-4     3000000               432 ns/op          32.36 MB/s          68 B/op          4 allocs/op
BenchmarkCodeUnmarshal/json-4                100          20498031 ns/op          94.67 MB/s     3274027 B/op      92663 allocs/op
BenchmarkCodeUnmarshal/phper-json-4           30          38771577 ns/op          50.05 MB/s    16434644 B/op     566562 allocs/op
BenchmarkUnmarshalString/json-4         10000000               181 ns/op             176 B/op          2 allocs/op
BenchmarkUnmarshalString/phper-json-4    2000000              1034 ns/op            2672 B/op          9 allocs/op
BenchmarkUnmarshalFloat64/json-4        10000000               163 ns/op             164 B/op          2 allocs/op
BenchmarkUnmarshalFloat64/phper-json-4   1000000              1120 ns/op            2660 B/op          9 allocs/op
BenchmarkUnmarshalInt64/json-4          10000000               124 ns/op             160 B/op          1 allocs/op
BenchmarkUnmarshalInt64/phper-json-4     2000000               983 ns/op            2656 B/op          8 allocs/op
BenchmarkUnmapped/json-4                 2000000               617 ns/op             216 B/op          4 allocs/op
BenchmarkUnmapped/phper-json-4            500000              2321 ns/op            2528 B/op         33 allocs/op
PASS
ok      github.com/shogo82148/go-phper-json     23.683s
```
