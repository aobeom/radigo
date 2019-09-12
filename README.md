# radigo
下载timeshift的音频，IP限制请自行解决，提供proxy参数使用。

## 使用说明

URL: http://radiko.jp/#!/ts/LFR/20190101005300

```go
Usage of radigo:
  -end_at string
        End DateTime eg: 20190101120000
  -proxy string
        Set a Proxy eg: 127.0.0.1:1080
  -start_at string
        Start DateTime eg: 20190101110000
  -station string
        Station ID eg: TBS
  -thread
        Thread Mode

go run radigo.go -station LFR -start_at 20190101005300 -end_at 20190101005800 -proxy 127.0.0.1:1080 -thread
```
