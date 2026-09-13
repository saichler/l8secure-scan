/*
(c) 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

package ifs

import (
	"fmt"
	"os"
)

var ANTHROPIC_API_KEY string

func init() {
	ANTHROPIC_API_KEY = os.Getenv("ANTHROPIC_API_KEY")
	if ANTHROPIC_API_KEY == "" {
		fmt.Println("[ifs] WARNING: ANTHROPIC_API_KEY environment variable is not set. AI Agent chat will not work.")
	}
}
