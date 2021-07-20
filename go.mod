module github.com/touchuyht/gortsplib

go 1.14

require (
	github.com/aler9/gortsplib v0.0.0-20210704172859-91ee859c9bf6
	github.com/icza/bitio v1.0.0
	github.com/pion/rtcp v1.2.4
	github.com/pion/rtp v1.6.1
	github.com/pion/sdp/v3 v3.0.2
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b
)

replace github.com/aler9/gortsplib v0.0.0-20210704172859-91ee859c9bf6 => ./
