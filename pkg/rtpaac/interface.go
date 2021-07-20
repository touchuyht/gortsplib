package rtpaac

import "time"

type AudioConfig interface {
	Decode([]byte) error
}

type RTPEncoder interface {
	Encode(aus [][]byte, firstPTS time.Duration) ([][]byte, error)
}
