package refs

// Exported is a function that other code uses as a value.
func Exported() int { return 1 }

func apply(f func() int) int { return f() }

// Run passes Exported as a value, and calls apply directly.
func Run() int { return apply(Exported) }

// Table stores the function in a value.
var Table = []func() int{Exported}

// Type is an exported type.
type Type struct{}

func (Type) Method() {}
