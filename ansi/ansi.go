package ansi

import "fmt"

const (
	Clear        string = "\033[2J"
	Reset_cursor string = "\033[H"
	Highlight    string = "\033[7m"
	Reset_text   string = "\033[0m"
	Show_cursor  string = "\033[?25h"
	Hide_cursor  string = "\033[?25l"
	Red          string = "\033[31m"
	Bold         string = "\033[1m"
)

func Text_color(r int, g int, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func Background_color(r int, g int, b int) string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
}
