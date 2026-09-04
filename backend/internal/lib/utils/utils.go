package utils

import (
	"encoding/json"
	"fmt"
)

// function to print formatted json from structs, maps & slices
func PrintJSON(v interface{}) {
	j, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON !!", err)
	}
	fmt.Println("JSON:", string(j))
}
