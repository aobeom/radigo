package radio

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"radigo/utils"
	"regexp"
	"strconv"
	"strings"
)

// Global 全局参数
const (
	UserAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36"
	XRadikoDevice     = "pc"
	XRadikoUser       = "dummy_user"
	XRadikoApp        = "pc_html5"
	XRadikoAppVersion = "0.0.1"
	FullRegionFile    = "stations.xml"
)

// Params 请求参数
type Params struct {
	StationID string
	StartAt   string
	EndAt     string
	Ft        string
	To        string
	L         string
	RType     string
}

// RegionData 地区列表
type RegionData struct {
	XMLName  xml.Name `xml:"region"`
	Stations []struct {
		RegionName string `xml:"region_name,attr"`
		RegionID   string `xml:"region_id,attr"`
		Station    []struct {
			ID     string `xml:"id"`
			AreaID string `xml:"area_id"`
		} `xml:"station"`
	} `xml:"stations"`
}

// XMLRead 读取 XML 内容
func XMLRead(s string) (regions RegionData) {
	xmlPath := utils.LocalPath(s)
	xmlFile, err := ioutil.ReadFile(xmlPath)
	if err != nil {
		log.Fatal(fmt.Sprintf("%s Read Failed", xmlPath), err)
	}
	err = xml.Unmarshal(xmlFile, &regions)
	if err != nil {
		log.Fatal(err)
	}
	return regions
}

// RegionXML 解析地区的 XML 数据
func RegionXML(searchType string, keyword string) (result string) {
	regions := XMLRead(FullRegionFile)

	for _, reg := range regions.Stations {
		for _, aid := range reg.Station {
			if searchType == "id" {
				if aid.ID == keyword {
					result = aid.AreaID
				}
			} else if searchType == "region" {
				if aid.ID == keyword {
					result = reg.RegionName
				} else if aid.AreaID == keyword {
					result = reg.RegionName
				}
			} else if searchType == "name" {
				if aid.AreaID == keyword {
					result = aid.ID
				}
			}
		}
	}
	return
}

// EncodeKey 根据偏移长度生成 KEY
func EncodeKey(authkey string, offset int64, length int64) (partialkey string) {
	reader := strings.NewReader(authkey)
	buff := make([]byte, length)
	_, err := reader.ReadAt(buff, offset)
	if err != nil {
		log.Fatal(err)
	}
	partialkey = base64.StdEncoding.EncodeToString(buff)
	return
}

// FilterChunklist 提取播放地址
func FilterChunklist(playlist string) (url string) {
	regURLRule := regexp.MustCompile(`https\:\/\/.*?\.m3u8`)
	urlList := regURLRule.FindAllString(string(playlist), -1)
	url = urlList[0]
	return
}

// FilterAAC 提取 AAC 文件的地址
func FilterAAC(m3u8 string) (urls []string) {
	regURLRule := regexp.MustCompile(`https\:\/\/.*?\.aac`)
	urls = regURLRule.FindAllString(string(m3u8), -1)
	return
}

// GetJSKey 提取 JS 的密钥
func GetJSKey() (authkey string) {
	playerURL := "http://radiko.jp/apps/js/playerCommon.js"

	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	body := utils.Minireq.GetBody(playerURL, headers, nil)

	regKeyRule := regexp.MustCompile(`[0-9a-z]{40}`)
	authkeyMap := regKeyRule.FindAllString(string(body), -1)
	authkey = authkeyMap[0]
	return
}

// IPCheck 检查 IP 是否符合
func IPCheck() (result bool) {
	checkURL := "http://radiko.jp/area"

	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	body := utils.Minireq.GetBody(checkURL, headers, nil)

	regAreaRule := regexp.MustCompile(`[^"<> ][A-Z0-9]+`)
	ipinfo := strings.Join(regAreaRule.FindAllString(string(body), -1), " ")
	if strings.Index(ipinfo, "OUT") == 0 {
		log.Fatal("IP Forbidden: " + strings.Replace(ipinfo, "OUT ", "", -1))
		result = false
	} else {
		result = true
	}
	return
}

// Auth1 获取 token / offset / length
func Auth1() (token string, offset int64, length int64) {
	auth1URL := "https://radiko.jp/v2/api/auth1"

	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	headers.Add("x-radiko-device", XRadikoDevice)
	headers.Add("x-radiko-user", XRadikoUser)
	headers.Add("x-radiko-app", XRadikoApp)
	headers.Add("x-radiko-app-version", XRadikoAppVersion)
	res := utils.Minireq.GetRes(auth1URL, headers, nil)

	resHeader := res.Header
	token = resHeader.Get("X-Radiko-AuthToken")
	Keyoffset := resHeader.Get("X-Radiko-Keyoffset")
	Keylength := resHeader.Get("X-Radiko-Keylength")
	offset, _ = strconv.ParseInt(Keyoffset, 10, 64)
	length, _ = strconv.ParseInt(Keylength, 10, 64)
	return
}

// Auth2 获取地区代码
func Auth2(token string, partialkey string) (area string) {
	auth2URL := "https://radiko.jp/v2/api/auth2"

	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	headers.Add("x-radiko-device", XRadikoDevice)
	headers.Add("x-radiko-user", XRadikoUser)
	headers.Add("x-radiko-authtoken", token)
	headers.Add("x-radiko-partialkey", partialkey)
	body := utils.Minireq.GetBody(auth2URL, headers, nil)

	areaSplit := strings.Split(string(body), ",")
	area = areaSplit[0]
	return
}

// GetChunklist 提取 AAC 下载地址
func GetChunklist(url string, token string) (aacURLs []string) {
	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	headers.Add("X-Radiko-AuthToken", token)
	m3u8Body := utils.Minireq.GetBody(url, headers, nil)
	aacURLs = FilterAAC(string(m3u8Body))
	return
}

// GetAACList 获取 AAC 下载地址
func GetAACList(token string, areaid string, radioData *Params) (aacURLs []string) {
	// 检测目标地区是否和 IP 地址匹配
	yourID := RegionXML("id", radioData.StationID)
	if areaid != yourID {
		yourArea := RegionXML("region", radioData.StationID)
		realArea := RegionXML("region", areaid)
		log.Fatal("Area Forbidden: You want to access " + yourArea + ", but your IP is recognized as " + realArea + ".")
	}

	playlistURL := "https://radiko.jp/v2/api/ts/playlist.m3u8"

	headers := make(http.Header)
	headers.Add("User-Agent", UserAgent)
	headers.Add("X-Radiko-AuthToken", token)
	headers.Add("X-Radiko-AreaId", areaid)

	params := make(map[string]string)
	params["station_id"] = radioData.StationID
	params["start_at"] = radioData.StartAt
	params["ft"] = radioData.Ft
	params["end_at"] = radioData.EndAt
	params["to"] = radioData.To
	params["l"] = radioData.L
	params["type"] = radioData.RType

	chunklistRes := utils.Minireq.GetRes(playlistURL, headers, params)

	code := chunklistRes.StatusCode
	if code == 200 {
		chunklistBody, _ := ioutil.ReadAll(chunklistRes.Body)
		chunklistURL := FilterChunklist(string(chunklistBody))

		m3u8headers := make(http.Header)
		m3u8headers.Add("User-Agent", UserAgent)
		m3u8Body := utils.Minireq.GetBody(chunklistURL, m3u8headers, nil)
		aacURLs = FilterAAC(string(m3u8Body))
	} else {
		ipBody := utils.Minireq.GetBody("http://whatismyip.akamai.com", nil, nil)
		realArea := RegionXML("region", areaid)
		log.Fatal("Area Forbidden: Your IP (" + string(ipBody) + ") is not in the area (" + realArea + ")")
	}
	return aacURLs
}
