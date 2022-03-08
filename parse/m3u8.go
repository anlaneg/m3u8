// Partial reference https://github.com/grafov/m3u8/blob/master/reader.go
package parse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type (
	PlaylistType string
	CryptMethod  string
)

const (
	PlaylistTypeVOD   PlaylistType = "VOD"
	PlaylistTypeEvent PlaylistType = "EVENT"

	CryptMethodAES  CryptMethod = "AES-128"
	CryptMethodNONE CryptMethod = "NONE"
)

// regex pattern for extracting `key=value` parameters from a line
var linePattern = regexp.MustCompile(`([a-zA-Z-]+)=("[^"]+"|[^",]+)`)

type M3u8 struct {
	Version        int8   // EXT-X-VERSION:version
	MediaSequence  uint64 // Default 0, #EXT-X-MEDIA-SEQUENCE:sequence
	Segments       []*Segment
	MasterPlaylist []*MasterPlaylist
	Keys           map[int]*Key
	EndList        bool         // #EXT-X-ENDLIST
	PlaylistType   PlaylistType // VOD or EVENT
	TargetDuration float64      // #EXT-X-TARGETDURATION:duration
}

type Segment struct {
	URI      string
	KeyIndex int
	Title    string  // #EXTINF: duration,<title>
	Duration float32 // #EXTINF: duration,<title>
	Length   uint64  // #EXT-X-BYTERANGE: length[@offset]
	Offset   uint64  // #EXT-X-BYTERANGE: length[@offset]
}

// #EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=240000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
type MasterPlaylist struct {
	URI        string
	BandWidth  uint32
	Resolution string
	Codecs     string
	ProgramID  uint32
}

// #EXT-X-KEY:METHOD=AES-128,URI="key.key"
type Key struct {
	// 'AES-128' or 'NONE'
	// If the encryption method is NONE, the URI and the IV attributes MUST NOT be present
	Method CryptMethod
	URI    string
	IV     string
}

func parse(reader io.Reader) (*M3u8, error) {
	s := bufio.NewScanner(reader)
	/*收集各行内容*/
	var lines []string
	for s.Scan() {
		lines = append(lines, s.Text())
	}

	var (
		i     = 0
		/*行数*/
		count = len(lines)
		m3u8  = &M3u8{
			Keys: make(map[int]*Key),
		}
		keyIndex = 0

		key     *Key
		seg     *Segment
		extInf  bool
		extByte bool
	)

	for ; i < count; i++ {
		line := strings.TrimSpace(lines[i])
		if i == 0 {
		    /*首行必须为#EXTM3U，且忽略*/
			if "#EXTM3U" != line {
				return nil, fmt.Errorf("invalid m3u8, missing #EXTM3U in line 1")
			}
			continue
		}
		switch {
		case line == "":
		    /*忽略空行*/
			continue
		case strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:"):
		    /*解析play list type且进行校验*/
			if _, err := fmt.Sscanf(line, "#EXT-X-PLAYLIST-TYPE:%s", &m3u8.PlaylistType); err != nil {
				return nil, err
			}
			isValid := m3u8.PlaylistType == "" || m3u8.PlaylistType == PlaylistTypeVOD || m3u8.PlaylistType == PlaylistTypeEvent
			if !isValid {
				return nil, fmt.Errorf("invalid playlist type: %s, line: %d", m3u8.PlaylistType, i+1)
			}
		case strings.HasPrefix(line, "#EXT-X-TARGETDURATION:"):
		    /*解析target duration*/
			if _, err := fmt.Sscanf(line, "#EXT-X-TARGETDURATION:%f", &m3u8.TargetDuration); err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
		    /*解析media sequence*/
			if _, err := fmt.Sscanf(line, "#EXT-X-MEDIA-SEQUENCE:%d", &m3u8.MediaSequence); err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, "#EXT-X-VERSION:"):
		    /*解析version*/
			if _, err := fmt.Sscanf(line, "#EXT-X-VERSION:%d", &m3u8.Version); err != nil {
				return nil, err
			}
		// Parse master playlist
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:"):
		    /*解析stream-inf*/
			mp, err := parseMasterPlaylist(line)
			if err != nil {
				return nil, err
			}
			/*下一行为uri*/
			i++
			mp.URI = lines[i]

			/*uri不得为空或者以'#'号开头*/
			if mp.URI == "" || strings.HasPrefix(mp.URI, "#") {
				return nil, fmt.Errorf("invalid EXT-X-STREAM-INF URI, line: %d", i+1)
			}
			/*添加解析生成的mp*/
			m3u8.MasterPlaylist = append(m3u8.MasterPlaylist, mp)

			//？？？？
			continue
		case strings.HasPrefix(line, "#EXTINF:"):
			if extInf {
			    /*只能出现一次*/
				return nil, fmt.Errorf("duplicate EXTINF: %s, line: %d", line, i+1)
			}
			if seg == nil {
				seg = new(Segment)
			}

			/*解出参数*/
			var s string
			if _, err := fmt.Sscanf(line, "#EXTINF:%s", &s); err != nil {
				return nil, err
			}

			/*如果s包含,号，则首先是title*/
			if strings.Contains(s, ",") {
				split := strings.Split(s, ",")
				seg.Title = split[1]
				s = split[0]
			}

			/*解析duration*/
			df, err := strconv.ParseFloat(s, 32)
			if err != nil {
				return nil, err
			}
			seg.Duration = float32(df)

			/*按顺序获得keyIndex*/
			seg.KeyIndex = keyIndex
			extInf = true
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
		    /*byte range只能出现一次*/
			if extByte {
				return nil, fmt.Errorf("duplicate EXT-X-BYTERANGE: %s, line: %d", line, i+1)
			}
			if seg == nil {
				seg = new(Segment)
			}
			/*解出byte range参数*/
			var b string
			if _, err := fmt.Sscanf(line, "#EXT-X-BYTERANGE:%s", &b); err != nil {
				return nil, err
			}

			/*不能为空*/
			if b == "" {
				return nil, fmt.Errorf("invalid EXT-X-BYTERANGE, line: %d", i+1)
			}

			/*如果包含@符，则前半部分为offset*/
			if strings.Contains(b, "@") {
				split := strings.Split(b, "@")
				offset, err := strconv.ParseUint(split[1], 10, 64)
				if err != nil {
					return nil, err
				}
				seg.Offset = uint64(offset)
				b = split[0]
			}

			/*解析seg对应的length*/
			length, err := strconv.ParseUint(b, 10, 64)
			if err != nil {
				return nil, err
			}
			seg.Length = uint64(length)
			extByte = true
		// Parse segments URI
		case !strings.HasPrefix(line, "#"):
		    /*遇到不能‘#’开头的行，如果前面遇到过EXTINF,即为此seg对应的uri，则添加segments*/
			if extInf {
				if seg == nil {
					return nil, fmt.Errorf("invalid line: %s", line)
				}
				/*记录此seg对应的uri*/
				seg.URI = line
				extByte = false
				extInf = false

				/*添加segments*/
				m3u8.Segments = append(m3u8.Segments, seg)
				seg = nil
				continue
			}
		// Parse key
		case strings.HasPrefix(line, "#EXT-X-KEY"):
		    /*解析key line*/
			params := parseLineParameters(line)
			if len(params) == 0 {
				return nil, fmt.Errorf("invalid EXT-X-KEY: %s, line: %d", line, i+1)
			}

			/*检查加密方法*/
			method := CryptMethod(params["METHOD"])
			if method != "" && method != CryptMethodAES && method != CryptMethodNONE {
				return nil, fmt.Errorf("invalid EXT-X-KEY method: %s, line: %d", method, i+1)
			}

			/*记录key*/
			keyIndex++
			key = new(Key)
			key.Method = method
			key.URI = params["URI"]
			key.IV = params["IV"]
			m3u8.Keys[keyIndex] = key
		case line == "#EndList":
		    /*标明list终止*/
			m3u8.EndList = true
		default:
		    /*忽略不认识的行*/
			continue
		}
	}

	return m3u8, nil
}

/*解析line,获得一组key,value对，并遍历这些kv对，填充mp*/
func parseMasterPlaylist(line string) (*MasterPlaylist, error) {
	params := parseLineParameters(line)
	if len(params) == 0 {
		return nil, errors.New("empty parameter")
	}
	mp := new(MasterPlaylist)
	for k, v := range params {
		switch {
		case k == "BANDWIDTH":
		    /*解析band width*/
			v, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, err
			}
			mp.BandWidth = uint32(v)
		case k == "RESOLUTION":
		    /*解析resolution*/
			mp.Resolution = v
		case k == "PROGRAM-ID":
		    /*解析program-id*/
			v, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, err
			}
			mp.ProgramID = uint32(v)
		case k == "CODECS":
		    /*解析codecs*/
			mp.Codecs = v

			/*忽略了不认识的key*/
		}
	}
	return mp, nil
}

// parseLineParameters extra parameters in string `line`
func parseLineParameters(line string) map[string]string {
    /*解析参数行，返回params*/
	r := linePattern.FindAllStringSubmatch(line, -1)
	params := make(map[string]string)
	for _, arr := range r {
		params[arr[1]] = strings.Trim(arr[2], "\"")
	}
	return params
}
