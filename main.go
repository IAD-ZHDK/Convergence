package main

import "os"

func main() {
	confluence := NewConfluence(
		os.Getenv("BASE_URL"),
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"),
	)

	convergence := NewConvergence(
		confluence,
		os.Getenv("HOME_SPACE_KEY"),
		os.Getenv("HOME_PAGE_TITLE"),
	)

	convergence.Run()
}
