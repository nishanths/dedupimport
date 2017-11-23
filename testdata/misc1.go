package p

import (
	`fmt`
	. "fmt"
	f "fmt"
)

import m `math` // to do math things
import fm "fmt"
import l "math"

import (
	"math"
)

import (
	w "math"
	x "math"
)

import (
	y "math"
	z "math"
)

// will fail to compile since the name l is repeated,
// but we need to fmt this correctly regardless.
import l "math"

import t "math"
import t "encoding/json"

import tt "math"
import tt "crypto/sha1"
import tt "crypto/sha256"

var fd int

func (b *Bar) method(argc, _ int, argv ...string) bool {
	x := true
	return x
}

type Bar struct {}

func foo(xxxxxx int, fmt string, /*f string*/) {
	_ = f
	_ = fmt.Println
	_ = Printf
	_ = f.Println
	_ = m.Sin
	_ = fm.Print
	_ = math.Cos
	_, _, _, _ = w.Tan, x.Tan, y.Tan, z.Tan

	type X struct {}

	lit := func(a int, enc json.Encoder) bool { return true }

	{
		var f int
		var math int
		_ = math.Sin
		titanic := 3
		type T struct {}
		_ = t.Sin
	}
}
