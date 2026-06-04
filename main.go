package main

import (
	"fmt"
	"ti_forge/convert"
)

func main() {
	var program_data []byte = convert.Read8xp("C:/Users/Jack/Onedrive/Desktop/go_work/ti_forge/TEST.8xp")
	display_data(program_data)
}

func display_data(program_data []byte) {
	fmt.Println(convert.Data_to_strings(program_data))
}
