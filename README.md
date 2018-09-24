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