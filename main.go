package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"golang.org/x/term"
	"ti_forge/convert"
	"unicode/utf8"
)

var cursor_row int = 1
var cursor_col int = 1

func main() {
	enable_utf8()
	
	fd := int(os.Stdin.Fd())
	cooked_state, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal("Failed to enter raw mode: ", err)
	}
	defer term.Restore(fd, cooked_state)

	var program_data []byte = convert.Read8xp("C:/Users/Jack/Onedrive/Desktop/go_work/ti_forge/TEST3.8xp")
	program_data_commands := convert.Data_to_strings(program_data)
	for {
		display_data(program_data_commands, cursor_row, cursor_col)
		switch process_input(get_input()) {
			case 1: return
		}
	}
}

func display_data(program_data_commands []string, input_cursor_row int, input_cursor_col int) {
	var program_data_lines [][]string
	var line_arr []string = []string{}
	
	for _, command := range program_data_commands {
		if command == "\n" {
			line_arr = append(line_arr, " ")
			program_data_lines = append(program_data_lines, line_arr)
			line_arr = []string{}
		} else {
			line_arr = append(line_arr, command)
		}
	}
	if len(line_arr) > 0 {
		program_data_lines = append(program_data_lines, line_arr)
	}

	if input_cursor_row < 1 {
		cursor_row = 1
		input_cursor_row = 1
	}
	if input_cursor_row > len(program_data_lines) {
		cursor_row = len(program_data_lines)
		input_cursor_row = cursor_row
	}
	if input_cursor_col < 1 {
		cursor_col = 1
		input_cursor_col = 1
	}
	var line_length int = len(program_data_lines[cursor_row-1])
	if input_cursor_col > line_length {
		cursor_col = line_length
		input_cursor_col = cursor_col
	}

	var program_data_line_strings []string = []string{}
	
	for _, line_arr := range program_data_lines {
		var line_builder strings.Builder
		for _, command := range line_arr {
			line_builder.WriteString(command)
		}
		program_data_line_strings = append(program_data_line_strings, line_builder.String())
	}

	max_line_num_len := len(strconv.Itoa(len(program_data_line_strings)))
	line_num_fmtstr := fmt.Sprintf("%%%ds %%s", max_line_num_len)

	fmt.Print("\033[H\033[2J")

	_, height := get_term_size()
	half_height := height/2
	var buffer_row int = cursor_row-1
	var buffer_start int = max(0, buffer_row-half_height)
	var buffer_end int = min(buffer_row+half_height, len(program_data_line_strings))

	var builder strings.Builder

	for i := buffer_start; i < buffer_end; i++ {
		fmt.Fprintf(&builder, line_num_fmtstr+"\n", strconv.Itoa(i+1), program_data_line_strings[i])
	}
	fmt.Print(builder.String())

	var screen_row int = buffer_row-buffer_start
	var screen_col int = max_line_num_len+2
	for i := 0; i < cursor_col-1; i++ {
    screen_col += utf8.RuneCountInString(program_data_lines[cursor_row-1][i])
	}

	fmt.Printf("\033[%d;%dH", screen_row+1, screen_col) // move cursor
}

func get_input() byte {
    buf := make([]byte, 1)
    n, err := os.Stdin.Read(buf)
    if err != nil || n == 0 {
        return 0
    }
    return buf[0]
}

func process_input(input byte) int {
	switch input {
	case 104: // Left
		cursor_col--
	case 108: // Right
		cursor_col++
	case 106: // Down
		cursor_row++
	case 107: // Up
		cursor_row--
	case 113: // "q"
		return 1
	}
	return 0
}

func get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}

func enable_utf8() {
	cmd := exec.Command("cmd.exe", "/c", "chcp 65001 > nul")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

