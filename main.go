package main

import (
    "encoding/base64"
    "encoding/xml"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
    "reflect"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"

    "golang.org/x/net/proxy"
)

// 参数解析
var pStationID = flag.String("station", "", "Station ID eg: TBS")
var pStartAt = flag.String("start_at", "", "Start DateTime eg: 20190101110000")
var pEndAt = flag.String("end_at", "", "End DateTime eg: 20190101120000")
var pProxy = flag.String("proxy", "", "Set a Proxy eg: 127.0.0.1:1080")
var pThread = flag.Bool("thread", false, "Thread Mode")

// Common Header 公共头
const (
    UserAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.132 Safari/537.36"
    XRadikoDevice     = "pc"
    XRadikoUser       = "dummy_user"
    XRadikoApp        = "pc_html5"
    XRadikoAppVersion = "0.0.1"
)

// DLServer 下载控制器
type DLServer struct {
    WG    sync.WaitGroup
    Gonum chan string
}

// RadioData 请求参数
type RadioData struct {
    stationID string
    startAt   string
    endAt     string
    ft        string
    to        string
    l         string
    rtype     string
}

// radikoRegion Region 地区列表
type radikoRegion struct {
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

// checkType 检查数据类型
func checkType(value interface{}) {
    fmt.Println(reflect.TypeOf(value))
}

// makePath 始终在当前目录
func makePath(path string) (newPath string) {
    workDir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    newPath = workDir + "//" + path
    return
}

// regionXML id / region / name
func regionXML(searchType string, keyword string) (result string) {
    xmlFile, err := ioutil.ReadFile(makePath("stations.xml"))
    if err != nil {
        log.Fatal(err)
    }
    var regions radikoRegion
    err = xml.Unmarshal(xmlFile, &regions)
    if err != nil {
        log.Fatal(err)
    }
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

// s5Proxy 设置 socket5 代理
func s5Proxy(proxyURL string) (transport *http.Transport) {
    dialer, err := proxy.SOCKS5("tcp", proxyURL,
        nil,
        &net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        },
    )
    if err != nil {
        log.Fatal("Proxy Error")
    }
    transport = &http.Transport{
        Proxy:               nil,
        Dial:                dialer.Dial,
        TLSHandshakeTimeout: 10 * time.Second,
    }
    return
}

// httpClient http 客户端
func httpClient(proxy string) (client http.Client) {
    client = http.Client{Timeout: 30 * time.Second}
    if proxy != "" {
        transport := s5Proxy(proxy)
        client = http.Client{Timeout: 30 * time.Second, Transport: transport}
    }
    return
}

// encodeKey 根据偏移长度生成 KEY
func encodeKey(authkey string, offset int64, length int64) (partialkey string) {
    reader := strings.NewReader(authkey)
    buff := make([]byte, length)
    _, err := reader.ReadAt(buff, offset)
    if err != nil {
        log.Fatal(err)
    }
    partialkey = base64.StdEncoding.EncodeToString(buff)
    return
}

// rProgress 进度条
func rProgress(i int, amp float64) {
    progress := float64(i) * amp
    num := int(progress / 10)
    if num < 1 {
        num = 1
    }
    pstyle := strings.Repeat(">", num)
    log.Printf("%s [%.2f %%]\r", pstyle, progress)
}

// radikoJSKey 提取 JS 的密钥
func radikoJSKey(client http.Client) (authkey string) {
    playerURL := "http://radiko.jp/apps/js/playerCommon.js"
    req, _ := http.NewRequest("GET", playerURL, nil)
    req.Header.Add("User-Agent", UserAgent)
    res, _ := client.Do(req)
    body, _ := ioutil.ReadAll(res.Body)
    regKeyRule := regexp.MustCompile(`[0-9a-z]{40}`)
    authkeyMap := regKeyRule.FindAllString(string(body), -1)
    authkey = authkeyMap[0]
    return
}

// radikoChunklist 提取播放地址
func radikoChunklist(playlist string) (url string) {
    regURLRule := regexp.MustCompile(`https\:\/\/.*?\.m3u8`)
    urlList := regURLRule.FindAllString(string(playlist), -1)
    url = urlList[0]
    return
}

// radikoAAC 提取 AAC 文件的地址
func radikoAAC(m3u8 string) (urls []string) {
    regURLRule := regexp.MustCompile(`https\:\/\/.*?\.aac`)
    urls = regURLRule.FindAllString(string(m3u8), -1)
    return
}

// radikoIPCheck 检查 IP 是否符合
func radikoIPCheck(client http.Client) (result bool) {
    checkURL := "http://radiko.jp/area"
    req, _ := http.NewRequest("GET", checkURL, nil)
    req.Header.Add("User-Agent", UserAgent)
    res, err := client.Do(req)
    if err != nil {
        log.Fatal("Plase Check Proxy")
    }
    body, _ := ioutil.ReadAll(res.Body)
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

// radikoAuth1 获取 token / offset / length
func radikoAuth1(client http.Client) (token string, offset int64, length int64) {
    auth1URL := "https://radiko.jp/v2/api/auth1"
    req, _ := http.NewRequest("GET", auth1URL, nil)
    req.Header.Add("User-Agent", UserAgent)
    req.Header.Add("x-radiko-device", XRadikoDevice)
    req.Header.Add("x-radiko-user", XRadikoUser)
    req.Header.Add("x-radiko-app", XRadikoApp)
    req.Header.Add("x-radiko-app-version", XRadikoAppVersion)
    res, _ := client.Do(req)
    header := res.Header
    token = header.Get("X-Radiko-AuthToken")
    Keyoffset := header.Get("X-Radiko-Keyoffset")
    Keylength := header.Get("X-Radiko-Keylength")
    offset, _ = strconv.ParseInt(Keyoffset, 10, 64)
    length, _ = strconv.ParseInt(Keylength, 10, 64)
    return
}

// radikoAuth2 获取地区代码
func radikoAuth2(client http.Client, token string, partialkey string) (area string) {
    auth2URL := "https://radiko.jp/v2/api/auth2"
    req, _ := http.NewRequest("GET", auth2URL, nil)
    req.Header.Add("User-Agent", UserAgent)
    req.Header.Add("x-radiko-device", XRadikoDevice)
    req.Header.Add("x-radiko-user", XRadikoUser)
    req.Header.Add("x-radiko-authtoken", token)
    req.Header.Add("x-radiko-partialkey", partialkey)
    res, _ := client.Do(req)
    body, _ := ioutil.ReadAll(res.Body)
    areaSplit := strings.Split(string(body), ",")
    area = areaSplit[0]
    return
}

// radikoHLS 获取 AAC 下载地址
func radikoHLS(client http.Client, token string, areaid string, radioData *RadioData) (aacURLs []string) {
    yourID := regionXML("id", radioData.stationID)
    if areaid != yourID {
        yourArea := regionXML("region", radioData.stationID)
        realArea := regionXML("region", areaid)
        log.Fatal("Area Forbidden: You want to access " + yourArea + ", but your IP is recognized as " + realArea + ".")
    }
    playlistURL := "https://radiko.jp/v2/api/ts/playlist.m3u8"
    req, _ := http.NewRequest("GET", playlistURL, nil)
    req.Header.Add("User-Agent", UserAgent)
    req.Header.Add("X-Radiko-AuthToken", token)
    req.Header.Add("X-Radiko-AreaId", areaid)
    params := req.URL.Query()
    params.Add("station_id", radioData.stationID)
    params.Add("start_at", radioData.startAt)
    params.Add("ft", radioData.ft)
    params.Add("end_at", radioData.endAt)
    params.Add("to", radioData.to)
    params.Add("l", radioData.l)
    params.Add("type", radioData.rtype)
    req.URL.RawQuery = params.Encode()
    chunklistRes, _ := client.Do(req)
    code := chunklistRes.StatusCode
    if code == 200 {
        chunklistBody, _ := ioutil.ReadAll(chunklistRes.Body)
        chunklistURL := radikoChunklist(string(chunklistBody))
        m3u8Req, _ := http.NewRequest("GET", chunklistURL, nil)
        m3u8Req.Header.Add("User-Agent", UserAgent)
        m3u8Res, _ := client.Do(m3u8Req)
        m3u8Body, _ := ioutil.ReadAll(m3u8Res.Body)
        aacURLs = radikoAAC(string(m3u8Body))
    } else {
        req, _ := http.NewRequest("GET", "http://whatismyip.akamai.com", nil)
        res, _ := client.Do(req)
        body, _ := ioutil.ReadAll(res.Body)
        realArea := regionXML("region", areaid)
        log.Fatal("Area Forbidden: Your IP (" + string(body) + ") is not in the area (" + realArea + ")")
    }
    return aacURLs
}

// rThread 并发下载
func rThread(client http.Client, url string, ch chan []byte, dl *DLServer) {
    req, _ := http.NewRequest("GET", url, nil)
    res, _ := client.Do(req)
    body, _ := ioutil.ReadAll(res.Body)
    dl.WG.Done()
    <-dl.Gonum
    ch <- body
}

// rEngine 下载器
func rEngine(client http.Client, mode string, urls []string, savePath string) {
    aacFile, _ := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    total := float64(len(urls))
    part, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", 100.0/total), 64)
    if mode == "single" {
        for i, url := range urls {
            req, _ := http.NewRequest("GET", url, nil)
            req.Header.Add("User-Agent", UserAgent)
            res, err := client.Do(req)
            if err != nil {
                log.Fatal("Request Error")
            }
            body, _ := ioutil.ReadAll(res.Body)
            offset, _ := aacFile.Seek(0, os.SEEK_END)
            aacFile.WriteAt(body, offset)
            rProgress(i, part)
        }
    } else if mode == "thread" {
        var thread int
        dl := new(DLServer)
        if total < 16 {
            thread = int(total)
        } else {
            thread = 16
        }
        var num int

        ch := make([]chan []byte, 1024)
        dl.Gonum = make(chan string, thread)
        dl.WG.Add(len(urls))

        for i, url := range urls {
            dl.Gonum <- url
            ch[i] = make(chan []byte)
            go rThread(client, url, ch[i], dl)
            rProgress(i, part)
        }
        for _, d := range ch {
            if num == int(total) {
                break
            }
            tmp, _ := <-d
            offset, _ := aacFile.Seek(0, os.SEEK_END)
            aacFile.WriteAt(tmp, offset)
            num++
        }

        dl.WG.Wait()
    }
    log.Println("Finished [ " + savePath + " ]")
    defer aacFile.Close()
}

// paraCheck 输入参数
func paraCheck() (radioData *RadioData) {
    flag.Parse()

    if *pStationID == "" {
        log.Fatal("[Error] Station ID. eg: TBS")
    }

    if *pStartAt == "" || len(*pStartAt) != 14 {
        log.Fatal("[Error] Start Date Time. eg: 20190101120000")
    }

    if *pEndAt == "" || len(*pEndAt) != 14 {
        log.Fatal("[Error] End Date Time. eg: 20190101130000")
    }

    if *pEndAt <= *pStartAt {
        log.Fatal("[Error] End > Start")
    }

    radioData = new(RadioData)
    radioData.stationID = *pStationID
    radioData.startAt = *pStartAt
    radioData.endAt = *pEndAt
    radioData.ft = *pStartAt
    radioData.to = *pEndAt
    radioData.l = "15"
    radioData.rtype = "b"
    return
}

func main() {
    radioData := paraCheck()

    var dMode string
    if *pThread {
        dMode = "thread"
    } else {
        dMode = "single"
    }

    start := time.Now()
    saveName := fmt.Sprintf("%s.%s.%s.aac", radioData.stationID, radioData.startAt, radioData.endAt)
    savePath := makePath(saveName)
    checkStation := regionXML("id", radioData.stationID)
    if checkStation == "" {
        log.Fatal("Station does not exist.")
    }
    client := httpClient(*pProxy)
    log.Println("Checking Your IP...")
    result := radikoIPCheck(client)
    if result == true {
        // log.Println("Get Key...")
        // authkey := radikoJSKey(client)
        authkey := "bcd151073c03b352e1ef2fd66c32209da9ca0afa"

        log.Println("Get Auth Info...")
        token, offset, length := radikoAuth1(client)
        partialkey := encodeKey(authkey, offset, length)
        log.Println("Get Radio Area...")
        area := radikoAuth2(client, token, partialkey)
        log.Println("Downloading...")
        aacURLs := radikoHLS(client, token, area, radioData)
        rEngine(client, dMode, aacURLs, savePath)
    }
    duration := time.Since(start)
    fmt.Println("Time: ", duration)
}
