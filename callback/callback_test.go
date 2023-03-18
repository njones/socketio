package callback

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/njones/socketio/serialize"
)

// eventCallback is just a wrapper function that will fake a callback. It's needed only for these examples.
func eventCallback(in ...interface{}) func(interface{ Callback(...interface{}) error }) error {
	return func(fn interface{ Callback(...interface{}) error }) error {
		return fn.Callback(in...)
	}
}

func ExampleErrorWrap() {
	err := eventCallback()(

		// This wraps a function that takes no arguments
		// and returns an error
		ErrorWrap(func() error {
			return fmt.Errorf("sad")
		}),
	)

	fmt.Println("err:", err)

	// Output: err: sad
}

func ExampleFuncString() {

	eventCallback("World")(

		// This wraps a function that takes a string as an argument and
		// and doesn't return
		FuncString(func(str string) {
			fmt.Println("Hello", str)
		}),
	)

	// Output: Hello World
}

func ExampleWrap() {

	// This can make any function callable without creating the interface

	err := eventCallback("Pan", "Wendy")(

		// Wrap takes in an object that represents a function
		// with parameters and an error output
		Wrap{
			Parameters: []serialize.Serializable{serialize.StrParam, serialize.StrParam},
			Func: func() interface{} {
				return func(last, first string) error {
					fmt.Println("Peter", last)
					fmt.Println(first, "Darling")
					return fmt.Errorf("Boys")
				}
			},
		},
	)

	fmt.Println("The Lost", err)

	eventCallback(1, "too", math.Pi, strings.NewReader("FORE"))(

		// Wrap takes in an object that represents a function
		// with parameters and an error output. Note how the
		// output of the function is a function that accepts
		// the parameters as arguments.
		Wrap{
			Parameters: []serialize.Serializable{
				serialize.IntParam, serialize.StrParam, serialize.F64Param, serialize.BinParam,
			},
			Func: func() interface{} {
				return func(one int, two string, three float64, four io.Reader) error {
					fmt.Println("The number:", one)
					fmt.Println("This takes:", two)
					fmt.Println("To make my:", three)
					fmt.Print("Stream out: ")
					_, err := io.Copy(os.Stdout, four)
					return err
				}
			},
		},
	)

	fmt.Println("\n====")

	eventCallback(map[string]interface{}{"Got 5 on it": "Luniz & Michael Marshall"})(

		// Wrap takes in an object that represents a function
		// with parameters and an error output. Note how the
		// output of the function is a function that accepts
		// the parameters as arguments.
		Wrap{
			Parameters: []serialize.Serializable{serialize.MapParam},
			Func: func() interface{} {
				return func(kv map[string]interface{}) error {
					for k, v := range kv {
						fmt.Printf("%s: %v\n", k, v)
					}
					return nil
				}
			},
		},
	)

	// Output:
	// Peter Pan
	// Wendy Darling
	// The Lost Boys
	// The number: 1
	// This takes: too
	// To make my: 3.141592653589793
	// Stream out: FORE
	// ====
	// Got 5 on it: Luniz & Michael Marshall
}

type CustomWrap func(string, string) error

func (cc CustomWrap) Callback(data ...interface{}) error {
	a, aOK := data[0].(string)
	b, bOK := data[1].(string)

	if !aOK || !bOK {
		return fmt.Errorf("bad parameters")
	}

	return cc(a, b)
}

func ExampleCustomWrap() {

	// This can make any function callable without creating the interface

	err := eventCallback("Pan", "Wendy")(CustomWrap(func(last, first string) error {
		fmt.Println("Peter", last)
		fmt.Println(first, "Darling")
		return nil
	}))

	fmt.Println("Error", err)

	// Output:
	// Peter Pan
	// Wendy Darling
	// Error <nil>
}

type CustomWrapMap func(map[string]interface{}) error

func (cc CustomWrapMap) Callback(data ...interface{}) error {
	a, aOK := data[0].(string)

	if !aOK {
		return fmt.Errorf("bad parameters")
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(a), &m); err != nil {
		return err
	}

	return cc(m)
}

func ExampleCustomWrapMap() {

	eventCallback(`{"pirate":"Captain Hook"}`)(CustomWrapMap(func(kv map[string]interface{}) error {
		if data, ok := kv["pirate"]; ok {
			fmt.Println(data)
		}
		return nil
	}))

	errParse := eventCallback(`not-a-map`)(CustomWrapMap(func(kv map[string]interface{}) error {
		if data, ok := kv["pirate"]; ok {
			fmt.Println(data)
		}
		return nil
	}))

	errCallback := eventCallback(`{"key":"value"}`)(CustomWrapMap(func(kv map[string]interface{}) error {
		if data, ok := kv["pirate"]; ok {
			fmt.Println(data)
			return nil
		}
		return fmt.Errorf("no key found")
	}))

	fmt.Println("Error:", errParse)
	fmt.Println("Error:", errCallback)

	// Output:
	// Captain Hook
	// Error: invalid character 'o' in literal null (expecting 'u')
	// Error: no key found
}
