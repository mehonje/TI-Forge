package ansi

const (
	Clear string = "\033[2J"
	Reset_cursor string = "\033[H"
	Highlight string = "\033[7m"
	Dehighlight string = "\033[0m"
	Show_cursor string = "\033[?25h"
	Hide_cursor string = "\033[?25l"
)
