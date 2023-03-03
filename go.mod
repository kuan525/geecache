module example

go 1.19

replace geecache => ./geecache

require geecache v0.0.0-00010101000000-000000000000

require (
	github.com/golang/protobuf v1.5.2 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
