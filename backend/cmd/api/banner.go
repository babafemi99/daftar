package main

import (
	"fmt"
	"os"

	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/logrusorgru/aurora/v4"
)

const (
	banner = `

░███████      ░███    ░██████████░██████████   ░███    ░█████████  
░██   ░██    ░██░██   ░██            ░██      ░██░██   ░██     ░██ 
░██    ░██  ░██  ░██  ░██            ░██     ░██  ░██  ░██     ░██ 
░██    ░██ ░█████████ ░█████████     ░██    ░█████████ ░█████████  
░██    ░██ ░██    ░██ ░██            ░██    ░██    ░██ ░██   ░██   
░██   ░██  ░██    ░██ ░██            ░██    ░██    ░██ ░██    ░██  
░███████   ░██    ░██ ░██            ░██    ░██    ░██ ░██     ░██ 
                                                                   
`
)

func printBanner(config cfg.Config) {
	colors := false
	if info, err := os.Stdout.Stat(); err == nil {
		colors = info.Mode()&os.ModeCharDevice != 0
	}
	color := aurora.New(aurora.WithColors(colors))

	fmt.Println(color.Bold(color.Cyan(banner)))
	fmt.Printf("%s %s\n", color.Bold("service:"), color.Green(config.ServiceName))
	fmt.Printf("%s %s\n", color.Bold("environment:"), color.Yellow(config.Environment))
	fmt.Printf("%s %s\n", color.Bold("listening:"), color.Green(config.HTTP.Address))
}
