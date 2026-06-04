package main

import (
	"fmt"
	"strconv"
	"strings"
	"ti_forge/convert"
)

func main() {
	var program_data []byte = convert.Read8xp("C:/Users/Jack/Onedrive/Desktop/go_work/ti_forge/TEST3.8xp")
	display_data(program_data)
}

func display_data(program_data_bytes []byte) {
	program_data_strings := convert.Data_to_strings(program_data_bytes)
	max_line_num_len := len(strconv.Itoa(len(program_data_strings)))
	line_num_fmtstr := fmt.Sprintf("%%%ds %%s", max_line_num_len)

	var builder strings.Builder
	builder.Grow(len(program_data_strings) * (max_line_num_len + 1 + 40))

	for i, line := range program_data_strings {
		fmt.Fprintf(&builder, line_num_fmtstr, strconv.Itoa(i), line)
	}
	fmt.Print(builder.String())
}
