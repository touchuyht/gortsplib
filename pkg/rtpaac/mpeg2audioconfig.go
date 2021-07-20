package rtpaac

import "fmt"

type ADTSHeader struct {
	//fixed header
	// 12 bit syncword '1111 1111 1111', denoting stary of a frame
	Syncword uint
	// 1 bit of MPEG tag, 0 for MPEG-4, 1 for MPEG-2
	Id uint
	// 2 bits which is always '00'
	Layer uint
	// 1 bit, 1 without crc and 0 indicated that crc is used
	ProtectionAbsent uint
	// 1 bit indicated which level of aac to use
	Profile uint
	// 4 bits represents sample rate
	SamplingFreIndex uint
	// 1 bit
	PrivateBit uint
	// 3 bit represents channel count
	ChannelCfg uint
	// 1 bit
	OriginalCopy uint
	// 1 bit
	Home uint

	// variable header
	// 1 bit
	CopyrightIdentificationBit uint
	// 1 bit
	CopyrightIndentificationStart uint
	// 13 bit represents size of the adts frame which includes header and audio stream
	AacFrameLength uint
	// 11 bit 0x7FF indicates that is a variable bitrate stream
	AdtsBufferFullness uint

	/* number_of_raw_data_blocks_in_frame
	 * 表示ADTS帧中有number_of_raw_data_blocks_in_frame + 1个AAC原始帧
	 * 所以说number_of_raw_data_blocks_in_frame == 0
	 * 表示说ADTS帧中有一个AAC数据块并不是说没有。(一个AAC原始帧包含一段时间内1024个采样及相关数据)
	 */
	// 2 bits
	NumberOfRawDataBlockInFrame uint
}

type MPEG2AudioConfig struct {
	Header ADTSHeader
}

// Decode decodes an MPEG_2 Audio Configuration
func (c *MPEG2AudioConfig) Decode(bytes []byte) error {
	if (bytes[0] == 0xFF) && ((bytes[1] & 0xF) == 0xF0) {
		c.Header.Id = (uint(bytes[1] & 0x08)) >> 3
		c.Header.Layer = (uint(bytes[1] & 0x06)) >> 1
		c.Header.ProtectionAbsent = uint(bytes[1] & 0x01)
		c.Header.Profile = (uint(bytes[2] & 0xc0)) >> 6
		c.Header.SamplingFreIndex = (uint(bytes[2] & 0x3c)) >> 2
		c.Header.PrivateBit = (uint(bytes[2] & 0x02)) >> 1
		c.Header.ChannelCfg = uint(((bytes[2] & 0x01) << 2) | ((bytes[3] & 0xc0) >> 6))
		c.Header.OriginalCopy = (uint(bytes[3] & 0x20)) >> 5
		c.Header.Home = (uint(bytes[3] & 0x10)) >> 4
		c.Header.CopyrightIdentificationBit = (uint(bytes[3] & 0x80)) >> 3
		c.Header.CopyrightIndentificationStart = (uint(bytes[3] & 0x40)) >> 2
		c.Header.AacFrameLength = (uint(bytes[3]&0x03) << 11) | (uint(bytes[4]) << 3) | (uint(bytes[5]&0xe0) >> 5)
		c.Header.AdtsBufferFullness = (uint(bytes[5]&0x1f) << 6) | (uint(bytes[6]&0xfc) >> 2)
		c.Header.NumberOfRawDataBlockInFrame = uint(bytes[6] & 0x03)
	} else {
		return fmt.Errorf("failed to decode config, err: incorrect format")
	}

	return nil
}
