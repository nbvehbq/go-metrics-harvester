package pkg2

import "os"

type dummy struct{}

func (d dummy) Run() int {
	return 0
}

func main() {
	m := dummy{}
	os.Exit(m.Run())
	os.Exit(1) // want "os.Exit call in main function"
}
