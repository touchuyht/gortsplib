package gortsplib

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	psdp "github.com/pion/sdp/v3"

	"github.com/aler9/gortsplib/pkg/base"
	"github.com/aler9/gortsplib/pkg/sdp"
	"github.com/touchuyht/gortsplib/pkg/rtpaac"
)

// Track is a RTSP track.
type Track struct {
	// attributes in SDP format
	Media *psdp.MediaDescription
}

func (t *Track) hasControlAttribute() bool {
	for _, attr := range t.Media.Attributes {
		if attr.Key == "control" {
			return true
		}
	}
	return false
}

// URL returns the track url.
func (t *Track) URL(baseURL *base.URL) (*base.URL, error) {
	if baseURL == nil {
		return nil, fmt.Errorf("empty base url")
	}

	controlAttr := func() string {
		for _, attr := range t.Media.Attributes {
			if attr.Key == "control" {
				return attr.Value
			}
		}
		return ""
	}()

	// no control attribute, use base URL
	if controlAttr == "" {
		return baseURL, nil
	}

	// control attribute contains an absolute path
	if strings.HasPrefix(controlAttr, "rtsp://") {
		ur, err := base.ParseURL(controlAttr)
		if err != nil {
			return nil, err
		}

		// copy host and credentials
		ur.Host = baseURL.Host
		ur.User = baseURL.User
		return ur, nil
	}

	// control attribute contains a relative control attribute
	// insert the control attribute at the end of the url
	// if there's a query, insert it after the query
	// otherwise insert it after the path
	strURL := baseURL.String()
	if controlAttr[0] != '?' && !strings.HasSuffix(strURL, "/") {
		strURL += "/"
	}
	ur, _ := base.ParseURL(strURL + controlAttr)
	return ur, nil
}

// ClockRate returns the clock rate of the track.
func (t *Track) ClockRate() (int, error) {
	if len(t.Media.MediaName.Formats) != 1 {
		return 0, fmt.Errorf("invalid format (%v)", t.Media.MediaName.Formats)
	}

	// get clock rate from payload type
	switch t.Media.MediaName.Formats[0] {
	case "0", "1", "2", "3", "4", "5", "7", "8", "9", "12", "13", "15", "18":
		return 8000, nil

	case "6":
		return 16000, nil

	case "10", "11":
		return 44100, nil

	case "14", "25", "26", "28", "31", "32", "33", "34":
		return 90000, nil

	case "16":
		return 11025, nil

	case "17":
		return 22050, nil
	}

	// get clock rate from rtpmap
	// https://tools.ietf.org/html/rfc4566
	// a=rtpmap:<payload type> <encoding name>/<clock rate> [/<encoding parameters>]
	for _, a := range t.Media.Attributes {
		if a.Key == "rtpmap" {
			tmp := strings.Split(a.Value, " ")
			if len(tmp) < 2 {
				return 0, fmt.Errorf("invalid rtpmap (%v)", a.Value)
			}

			tmp = strings.Split(tmp[1], "/")
			if len(tmp) != 2 && len(tmp) != 3 {
				return 0, fmt.Errorf("invalid rtpmap (%v)", a.Value)
			}

			v, err := strconv.ParseInt(tmp[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return int(v), nil
		}
	}
	return 0, fmt.Errorf("attribute 'rtpmap' not found")
}

// NewTrackH264 initializes an H264 track from a SPS and PPS.
func NewTrackH264(payloadType uint8, sps []byte, pps []byte) (*Track, error) {
	spropParameterSets := base64.StdEncoding.EncodeToString(sps) +
		"," + base64.StdEncoding.EncodeToString(pps)
	profileLevelID := strings.ToUpper(hex.EncodeToString(sps[1:4]))

	typ := strconv.FormatInt(int64(payloadType), 10)

	return &Track{
		Media: &psdp.MediaDescription{
			MediaName: psdp.MediaName{
				Media:   "video",
				Protos:  []string{"RTP", "AVP"},
				Formats: []string{typ},
			},
			Attributes: []psdp.Attribute{
				{
					Key:   "rtpmap",
					Value: typ + " H264/90000",
				},
				{
					Key: "fmtp",
					Value: typ + " packetization-mode=1; " +
						"sprop-parameter-sets=" + spropParameterSets + "; " +
						"profile-level-id=" + profileLevelID,
				},
			},
		},
	}, nil
}

// IsH264 checks whether the track is a H264 track.
func (t *Track) IsH264() bool {
	if t.Media.MediaName.Media != "video" {
		return false
	}

	v, ok := t.Media.Attribute("rtpmap")
	if !ok {
		return false
	}

	vals := strings.Split(v, " ")
	if len(vals) != 2 {
		return false
	}

	return vals[1] == "H264/90000"
}

// ExtractDataH264 extracts the SPS and PPS from an H264 track.
func (t *Track) ExtractDataH264() ([]byte, []byte, error) {
	v, ok := t.Media.Attribute("fmtp")
	if !ok {
		return nil, nil, fmt.Errorf("fmtp attribute is missing")
	}

	tmp := strings.SplitN(v, " ", 2)
	if len(tmp) != 2 {
		return nil, nil, fmt.Errorf("invalid fmtp attribute (%v)", v)
	}

	var sps []byte
	var pps []byte

	for _, kv := range strings.Split(tmp[1], ";") {
		kv = strings.Trim(kv, " ")

		if len(kv) == 0 {
			continue
		}

		tmp := strings.SplitN(kv, "=", 2)
		if len(tmp) != 2 {
			return nil, nil, fmt.Errorf("invalid fmtp attribute (%v)", v)
		}

		if tmp[0] == "sprop-parameter-sets" {
			tmp := strings.SplitN(tmp[1], ",", 2)
			if len(tmp) != 2 {
				return nil, nil, fmt.Errorf("invalid sprop-parameter-sets (%v)", v)
			}

			var err error
			sps, err = base64.StdEncoding.DecodeString(tmp[0])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid sprop-parameter-sets (%v)", v)
			}

			pps, err = base64.StdEncoding.DecodeString(tmp[1])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid sprop-parameter-sets (%v)", v)
			}
		}
	}

	if sps == nil || pps == nil {
		return nil, nil, fmt.Errorf("sprop-parameter-sets is missing (%v)", v)
	}

	return sps, pps, nil
}

// NewTrackAAC initializes an AAC track from a configuration.
func NewTrackAAC(payloadType uint8, config []byte) (*Track, error) {
	var (
		mpegStandard uint8
		mpeg4Conf    rtpaac.MPEG4AudioConfig
		mpeg2Conf    rtpaac.MPEG2AudioConfig
	)

	err := mpeg4Conf.Decode(config)
	if err != nil {
		err := mpeg2Conf.Decode(config)
		if err != nil {
			return nil, fmt.Errorf("unsupported audio track")
		}
		mpegStandard = 2
	} else {
		mpegStandard = 4
	}

	typ := strconv.FormatInt(int64(payloadType), 10)

	switch mpegStandard {
	case 2:
		var tempConfig uint16 = 0
		tempConfig = uint16(mpeg2Conf.Header.Profile + 1)
		tempConfig <<= 5
		tempConfig |= uint16(mpeg2Conf.Header.SamplingFreIndex)
		tempConfig <<= 4
		tempConfig |= uint16(mpeg2Conf.Header.ChannelCfg)
		tempConfig <<= 3
		return &Track{
			Media: &psdp.MediaDescription{
				MediaName: psdp.MediaName{
					Media:   "audio",
					Protos:  []string{"RTP", "AVP"},
					Formats: []string{typ},
				},
				Attributes: []psdp.Attribute{
					{
						Key: "rtpmap",
						Value: typ + " mpeg4-generic/" + strconv.FormatInt(int64(mpeg2Conf.Header.SamplingFreIndex), 10) +
							"/" + strconv.FormatInt(int64(mpeg2Conf.Header.ChannelCfg), 10),
					},
					{
						Key: "fmtp",
						Value: typ + " streamtype=5;profile-level-id=1; " +
							"mode=AAC-hbr; " +
							"sizelength=13; " +
							"indexlength=3; " +
							"indexdeltalength=3; " +
							"config=" + fmt.Sprintln("%x", tempConfig),
					},
				},
			},
		}, nil
	case 4:
		return &Track{
			Media: &psdp.MediaDescription{
				MediaName: psdp.MediaName{
					Media:   "audio",
					Protos:  []string{"RTP", "AVP"},
					Formats: []string{typ},
				},
				Attributes: []psdp.Attribute{
					{
						Key: "rtpmap",
						Value: typ + " mpeg4-generic/" + strconv.FormatInt(int64(mpeg4Conf.SampleRate), 10) +
							"/" + strconv.FormatInt(int64(mpeg4Conf.ChannelCount), 10),
					},
					{
						Key: "fmtp",
						Value: typ + " profile-level-id=1; " +
							"mode=AAC-hbr; " +
							"sizelength=13; " +
							"indexlength=3; " +
							"indexdeltalength=3; " +
							"config=" + hex.EncodeToString(config),
					},
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported audio track")
}

// IsAAC checks whether the track is an AAC track.
func (t *Track) IsAAC() bool {
	if t.Media.MediaName.Media != "audio" {
		return false
	}

	v, ok := t.Media.Attribute("rtpmap")
	if !ok {
		return false
	}

	vals := strings.Split(v, " ")
	if len(vals) != 2 {
		return false
	}

	return strings.HasPrefix(strings.ToLower(vals[1]), "mpeg4-generic/")
}

// ExtractDataAAC extracts the config from an AAC track.
func (t *Track) ExtractDataAAC() ([]byte, error) {
	v, ok := t.Media.Attribute("fmtp")
	if !ok {
		return nil, fmt.Errorf("fmtp attribute is missing")
	}

	tmp := strings.SplitN(v, " ", 2)
	if len(tmp) != 2 {
		return nil, fmt.Errorf("invalid fmtp (%v)", v)
	}

	var config []byte

	for _, kv := range strings.Split(tmp[1], ";") {
		kv = strings.Trim(kv, " ")

		if len(kv) == 0 {
			continue
		}

		tmp := strings.SplitN(kv, "=", 2)
		if len(tmp) != 2 {
			return nil, fmt.Errorf("invalid fmtp (%v)", v)
		}

		if tmp[0] == "config" {
			var err error
			config, err = hex.DecodeString(tmp[1])
			if err != nil {
				return nil, fmt.Errorf("invalid config (%v)", v)
			}
		}
	}

	if config == nil {
		return nil, fmt.Errorf("config is missing (%v)", v)
	}

	return config, nil
}

// Tracks is a list of tracks.
type Tracks []*Track

// ReadTracks decodes tracks from SDP.
func ReadTracks(byts []byte) (Tracks, error) {
	desc := sdp.SessionDescription{}
	err := desc.Unmarshal(byts)
	if err != nil {
		return nil, err
	}

	tracks := make(Tracks, len(desc.MediaDescriptions))

	for i, media := range desc.MediaDescriptions {
		tracks[i] = &Track{
			Media: media,
		}
	}

	// since ReadTracks is used to handle ANNOUNCE and SETUP requests,
	// all tracks must have a valid clock rate.
	for i, track := range tracks {
		_, err := track.ClockRate()
		if err != nil {
			return nil, fmt.Errorf("unable to get clock rate of track %d: %s", i, err)
		}
	}

	return tracks, nil
}

func cloneAndClearTracks(ts Tracks) Tracks {
	ret := make(Tracks, len(ts))

	for i, track := range ts {
		md := &psdp.MediaDescription{
			MediaName: psdp.MediaName{
				Media:   track.Media.MediaName.Media,
				Protos:  []string{"RTP", "AVP"}, // override protocol
				Formats: track.Media.MediaName.Formats,
			},
			Bandwidth: track.Media.Bandwidth,
			Attributes: func() []psdp.Attribute {
				var ret []psdp.Attribute

				for _, attr := range track.Media.Attributes {
					if attr.Key == "rtpmap" || attr.Key == "fmtp" {
						ret = append(ret, attr)
					}
				}

				ret = append(ret, psdp.Attribute{
					Key:   "control",
					Value: "trackID=" + strconv.FormatInt(int64(i), 10),
				})

				return ret
			}(),
		}

		ret[i] = &Track{
			Media: md,
		}
	}

	return ret
}

// Write encodes tracks into SDP.
func (ts Tracks) Write() []byte {
	sout := &sdp.SessionDescription{
		SessionName: psdp.SessionName("Stream"),
		Origin: psdp.Origin{
			Username:       "-",
			NetworkType:    "IN",
			AddressType:    "IP4",
			UnicastAddress: "127.0.0.1",
		},
		// required by Darwin Streaming Server
		ConnectionInformation: &psdp.ConnectionInformation{
			NetworkType: "IN",
			AddressType: "IP4",
			Address:     &psdp.Address{Address: "0.0.0.0"},
		},
		TimeDescriptions: []psdp.TimeDescription{
			{Timing: psdp.Timing{0, 0}}, //nolint:govet
		},
	}

	for _, track := range ts {
		sout.MediaDescriptions = append(sout.MediaDescriptions, track.Media)
	}

	byts, _ := sout.Marshal()
	return byts
}
