// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2018 Shogo Ichinose. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phperjson_test

import (
	"fmt"
	"os"
	"reflect"

	phperjson "github.com/shogo82148/go-phper-json"
)

func ExampleMarshal() {
	type ColorGroup struct {
		ID     int
		Name   string
		Colors []string
	}
	group := ColorGroup{
		ID:     1,
		Name:   "Reds",
		Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
	}
	// phperjson.Marshal is compatible with json.Marshal.
	b, err := phperjson.Marshal(group)
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(b)
	// Output:
	// {"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}
}

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
