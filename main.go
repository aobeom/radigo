package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"radigo/radio"
	"radigo/utils"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 参数解析
var pStationID = flag.String("station", "", "Station ID eg: TBS")
var pStartAt = flag.String("start_at", "", "Start DateTime eg: 20190101110000")
var pEndAt = flag.String("end_at", "", "End DateTime eg: 20190101120000")
var pProxy = flag.String("proxy", "", "Set a Proxy eg: 127.0.0.1:1080")
var pRawList = flag.String("raw", "", "Download AAC Only")

// DLServer 下载控制器
type DLServer struct {
	WG    sync.WaitGroup
	Gonum chan string
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

// rThread 并发下载
func rThread(url string, ch chan []byte, dl *DLServer) {
	headers := make(http.Header)
	headers.Add("User-Agent", radio.UserAgent)
	body := utils.Minireq.GetBody(url, headers, nil)
	dl.WG.Done()
	<-dl.Gonum
	ch <- body
}

// rEngine 下载器
func rEngine(thread bool, urls []string, savePath string) {
	aacFile, _ := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	total := float64(len(urls))
	part, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", 100.0/total), 64)
	if thread {
		var thread int
		dl := new(DLServer)
		if total < 16 {
			thread = int(total)
		} else {
			thread = 16
		}
		var num int

		ch := make([]chan []byte, 8192)
		dl.Gonum = make(chan string, thread)
		dl.WG.Add(len(urls))

		for i, url := range urls {
			dl.Gonum <- url
			ch[i] = make(chan []byte, 8192)
			go rThread(url, ch[i], dl)
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
	} else {
		for i, url := range urls {
			headers := make(http.Header)
			headers.Add("User-Agent", radio.UserAgent)
			body := utils.Minireq.GetBody(url, headers, nil)
			offset, _ := aacFile.Seek(0, os.SEEK_END)
			aacFile.WriteAt(body, offset)
			rProgress(i, part)
		}
	}
	log.Println("Finished [ " + savePath + " ]")
	defer aacFile.Close()
}

// setPara 设置参数
func setPara() (radioData *radio.Params, m3u8Raw string) {
	flag.Parse()

	if *pProxy != "" {
		utils.Proxy = *pProxy
	}

	if *pRawList == "" {
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

		radioData = new(radio.Params)
		radioData.StationID = *pStationID
		radioData.StartAt = *pStartAt
		radioData.EndAt = *pEndAt
		radioData.Ft = *pStartAt
		radioData.To = *pEndAt
		radioData.L = "15"
		radioData.RType = "b"
		m3u8Raw = ""
	} else {
		m3u8Raw = *pRawList
	}
	return
}

// StationChecker 检查电台编号
func StationChecker(i string) {
	checkStation := radio.RegionXML("id", i)
	if checkStation == "" {
		log.Fatal("Station does not exist.")
	}
}

func main() {
	radioData, rawFile := setPara()
	// 计算执行时间
	start := time.Now()

	if rawFile == "" {
		// 设置保存路径
		saveName := fmt.Sprintf("%s.%s.%s.aac", radioData.StationID, radioData.StartAt, radioData.EndAt)
		savePath := utils.LocalPath(saveName)

		StationChecker(radioData.StationID)
		log.Println("Checking Your IP...")
		result := radio.IPCheck()
		if result == true {
			// 1.获取 JS Key
			// log.Println("Get Key...")
			// authkey := radikoJSKey(client)
			authkey := "bcd151073c03b352e1ef2fd66c32209da9ca0afa"
			// 2.获取认证信息
			log.Println("Get Auth Info...")
			token, offset, length := radio.Auth1()
			partialkey := radio.EncodeKey(authkey, offset, length)
			// 3.获取播放地区
			log.Println("Get Radio Area...")
			area := radio.Auth2(token, partialkey)
			// 4.开始下载
			log.Println("Downloading...")
			aacURLs := radio.GetAACList(token, area, radioData)
			rEngine(true, aacURLs, savePath)
		}
	} else {
		saveName := fmt.Sprintf("%s.raw.acc", time.Now().Format("2006-01-02-15-04-05"))
		savePath := utils.LocalPath(saveName)
		rawData := utils.ReadFile(rawFile)
		aacURLs := radio.FilterAAC(rawData)
		msg := fmt.Sprintf("Total: %d", len(aacURLs))
		log.Println(msg)
		rEngine(true, aacURLs, savePath)
	}
	duration := time.Since(start)
	fmt.Println("Time: ", duration)
}
