package main

import "os"

func main() {
	confluence := NewConfluence()
	confluence.BaseURL = os.Getenv("BASE_URL")
	confluence.Username = os.Getenv("USERNAME")
	confluence.Password = os.Getenv("PASSWORD")

	convergence := NewConvergence()
	convergence.Confluence = confluence

	convergence.Run()
}
