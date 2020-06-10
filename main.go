package main

import "bronx.release/builder"

func main() {
	// release builder doesn't need to expose two function to consumer
	var releaseBuilder builder.ReleaseBuilder
	releaseBuilder.Initialize()
	releaseBuilder.Run()
}
