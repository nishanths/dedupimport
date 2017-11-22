package p

import (
	`fmt`
	. "fmt"
	f "fmt"
)

import m `math`
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

// var f int
	
func foo() {
	_ = f
	_ = fmt.Println
	_ = Printf
	_ = f.Println
	_ = m.Sin
	_ = fm.Print
	_ = math.Cos
	_, _, _, _ = w.Tan, x.Tan, y.Tan, z.Tan
}
