package fx

func Entry() int { return helper() + 1 }

func helper() int { return leaf() }

func leaf() int { return 1 }

type T struct{}

func (T) Method() int { return leaf() }
