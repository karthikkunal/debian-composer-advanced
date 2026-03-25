package main

import (
    "fmt"
    "os"
)

func main() {
    // Check if fragment file exists
    path := "../../debian-composer/kitchen/components/base.yaml"
    data, err := os.ReadFile(path)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Read", len(data), "bytes")
        // Print first 500 chars
        fmt.Println(string(data[:500]))
    }
}
