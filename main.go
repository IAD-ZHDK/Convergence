package main

import "os"

func main() {
	confluence := NewConfluence(
		os.Getenv("BASE_URL"),
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"),
	)

	convergence := NewConvergence(confluence)

	convergence.Run()
}
