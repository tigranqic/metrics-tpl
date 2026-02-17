package aliasshadow

type os struct{}

func (o os) Exit(code int) {}

func foo() {
	var myos os
	myos.Exit(1)
}
