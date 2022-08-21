package callback

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/njones/socketio/serialize"
)

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

	// Output:
	// Peter Pan
	// Wendy Darling
	// The Lost Boys
	// The number: 1
	// This takes: too
	// To make my: 3.141592653589793
	// Stream out: FORE
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
