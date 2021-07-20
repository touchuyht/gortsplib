package main

import (
	"fmt"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/rtpaac"
)

// This example shows how to
// 1. connect to a RTSP server and read all tracks on a path

func main() {
	// connect to the server and start reading all tracks
	conn, err := gortsplib.DialRead("rtsps://touchuyht.com:8554/mystream")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// find the aac track
	aacTrack := func() int {
		for i, track := range conn.Tracks() {
			if track.IsAAC() {
				return i
			}
		}
		return -1
	}()
	if aacTrack < 0 {
		panic(fmt.Errorf("Aac track not found"))
	}
	fmt.Printf("Aac track is number %d\n", aacTrack+1)

	dec := rtpaac.NewDecoder(16000)
	sequenceNumber := uint16(0x44ed)
	ssrc := uint32(0x9dbb7812)
	initialTs := uint32(0x88776655)
	enc := rtpaac.NewEncoder(96, 48000, &sequenceNumber, &ssrc, &initialTs)

	// read RTP frames
	err = conn.ReadFrames(func(trackID int, streamType gortsplib.StreamType, payload []byte) {
		if streamType == gortsplib.StreamTypeRTP && trackID == aacTrack {
			// convert RTP frames into aac NALUs
			nalus, _, err := dec.Decode(payload)
			if err != nil {
				return
			}

			// print NALUs
			for _, nalu := range nalus {
				fmt.Printf("received Aac NALU of size %d\n", len(nalu))
			}

			_ , err = enc.Encode(nalus, 2 * time.Millisecond)
			if err != nil {
				fmt.Println(err)
			}
		}
	})
	panic(err)

	// read RTP frames
	//err = conn.ReadFrames(func(trackID int, streamType gortsplib.StreamType, payload []byte) {
	//	fmt.Printf("frame from track %d, type %v, size %d\n", trackID, streamType, len(payload))
	//})
	//panic(err)

	// decode RTP to aac-lc
}
