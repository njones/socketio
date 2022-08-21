package serialize

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func ExampleSerializable_assignment() {
	// Serialized values can be assigned to a variable before being used

	var e = Error(fmt.Errorf("some bad thing"))
	var f = Float64(1.4)
	var i = Integer(10)
	var s = String("string")
	var u = Uinteger(11)

	fmt.Printf("Error: %v\nFloat: %f\nInt: %d\nString: %q\nUint: %d\n", e, f.Interface(), i.Interface(), s, u.Interface())

	// Output:
	// Error: some bad thing
	// Float: 1.400000
	// Int: 10
	// String: "string"
	// Uint: 11
}

func ExampleSerializable_short_func() {

	// Assign the serialize function to a short function name for later usage

	var er = Error
	var fl = Float64
	var in = Integer
	var st = String
	var ui = Uinteger

	var a, b, c = er(fmt.Errorf("error A")), er(fmt.Errorf("error B")), er(fmt.Errorf("error C"))
	var d, e, f = fl(1.5), fl(2.6), fl(3.7)
	var g, h, i = in(100), in(-100), in(1000000)
	var j, k, l = st("The"), st("Quick"), st("Brown")
	var m, n, o = ui(400), ui(800), ui(9223372036854775808)

	fmt.Printf("%v\n", a)
	fmt.Printf("%v\n", b)
	fmt.Printf("%v\n", c)
	fmt.Printf("%f\n", *d)
	fmt.Printf("%f\n", *e)
	fmt.Printf("%f\n", *f)
	fmt.Printf("%d\n", *g)
	fmt.Printf("%d\n", *h)
	fmt.Printf("%d\n", *i)
	fmt.Printf("%s\n", j)
	fmt.Printf("%s\n", k)
	fmt.Printf("%s\n", l)
	fmt.Printf("%d\n", *m)
	fmt.Printf("%d\n", *n)
	fmt.Printf("%d\n", *o)

	// Output:
	// error A
	// error B
	// error C
	// 1.500000
	// 2.600000
	// 3.700000
	// 100
	// -100
	// 1000000
	// The
	// Quick
	// Brown
	// 400
	// 800
	// 9223372036854775808
}

func ExampleSerializable_unserialize() {

	var e = Error(nil)
	var f = Float64(0)
	var i = Integer(0)
	var s = String("")
	var u = Uinteger(0)

	e.Unserialize("something bad happened")
	f.Unserialize("3.14")
	i.Unserialize("10")
	s.Unserialize("pass")
	u.Unserialize("20")

	es, _ := e.Serialize()
	fs, _ := f.Serialize()
	is, _ := i.Serialize()
	ss, _ := s.Serialize()
	us, _ := u.Serialize()

	fmt.Printf("%s\n%s\n%s\n%s\n%s", es, fs, is, ss, us)

	// Output:
	// something bad happened
	// 3.14
	// 10
	// pass
	// 20
}

func ExampleSerializable_string() {

	var e = Error(nil)
	var f = Float64(0)
	var i = Integer(0)
	var s = String("")
	var u = Uinteger(0)

	e.Unserialize("something bad happened")
	f.Unserialize("3.14")
	i.Unserialize("10")
	s.Unserialize("pass")
	u.Unserialize("20")

	fmt.Printf("%s\n%s\n%s\n%s\n%s", e, f, i, s, u)

	// Output:
	// something bad happened
	// 3.14
	// 10
	// pass
	// 20
}

func ExampleSerializable_interface() {

	var e = Error(nil)
	var f = Float64(0)
	var i = Integer(0)
	var s = String("")
	var u = Uinteger(0)

	e.Unserialize("something bad happened")
	f.Unserialize("3.14")
	i.Unserialize("10")
	s.Unserialize("pass")
	u.Unserialize("20")

	fmt.Printf("%v\n%f\n%d\n%q\n%d", e.Interface(), f.Interface(), i.Interface(), s.Interface(), u.Interface())

	// Output:
	// something bad happened
	// 3.140000
	// 10
	// "pass"
	// 20
}

func ExampleSerializable_params() {

	var e = ErrParam
	var f = F64Param
	var i = IntParam
	var s = StrParam
	var u = UintParam

	e.Unserialize("something bad happened")
	f.Unserialize("3.14")
	i.Unserialize("10")
	s.Unserialize("pass")
	u.Unserialize("20")

	fmt.Printf("%q\n%q\n%q\n%q\n%q", e, f, i, s, u)

	// Output:
	// ""
	// ""
	// ""
	// ""
	// ""
}

func ExampleSerializable_binary() {
	r := strings.NewReader("This is streamed data")

	// binary doesn't serialize to a string, but it
	// provides a io.Reader from the Interface() method
	b := Binary(r)

	io.Copy(os.Stdout, b.Interface().(io.Reader))
	b.Interface().(io.Seeker).Seek(0, 0)
	fmt.Println("\n----")
	// There is also the hidden (io.Reader) interface

	io.Copy(os.Stdout, b)

	// Output:
	// This is streamed data
	// ----
	// This is streamed data
}

type pizza struct {
	crust    string
	toppings []string
}

func (z *pizza) Serialize() (string, error) {
	return fmt.Sprintf("crust:%s top:%s", z.crust, strings.Join(z.toppings, ",")), nil
}
func (z *pizza) Unserialize(str string) error {
	items := strings.Split(str, " ")
	for i, item := range items {
		if i == 0 {
			z.crust = strings.TrimPrefix(item, "crust:")
			continue
		}
		z.toppings = strings.Split(strings.TrimPrefix(item, "top:"), ",")
	}
	return nil
}

func ExampleSerializable_custom() {

	pepperoni := &pizza{crust: "thin", toppings: []string{"cheese", "pepperoni"}}
	fmt.Println(pepperoni.Serialize())

	cheese := &pizza{}
	cheese.Unserialize("crust:cheese top:mozzarella,gorgonzola,goat,parmesan")
	fmt.Println(cheese)

	// Output:
	// crust:thin top:cheese,pepperoni <nil>
	// &{cheese [mozzarella gorgonzola goat parmesan]}
}
