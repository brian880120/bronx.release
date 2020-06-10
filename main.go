package main

import "bronx.release/builder"

func main() {
	var releaseBuilder builder.ReleaseBuilder
	releaseBuilder.Initialize()
	releaseBuilder.Run()
}
