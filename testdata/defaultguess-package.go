package fmt // my fmt package

import "fmt/v1" // package is called v1, but our default guessing will guess it incorrectly as "fmt"
import f "fmt/v1"

func foo() {
	v1.Println("1")

	// will try to rewrite this to f -> fmt (incorrectly; should be f -> v1)
	// but our package check will catch it and print an inability to rewrite error
	// instead.
	f.Println("2") 
}
