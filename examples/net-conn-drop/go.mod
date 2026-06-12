module github.com/RomanAgaltsev/chaotic/examples/net-conn-drop

go 1.26

toolchain go1.26.4

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/net v0.0.0
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/net => ../../adapter/net
