package main

import (
    "crypto/tls"
    "encoding/json"
    "fmt"
    "github.com/fzdwx/infinite/components"
    "github.com/fzdwx/infinite/components/progress"
    "github.com/fzdwx/infinite/components/spinner"
    "github.com/k0kubun/go-ansi"
    "github.com/schollz/progressbar/v3"
    "io"
    "net/http"
    "net/url"
    "os"
    "path"
    "regexp"
    "time"
)

type Release struct {
    Name   string   `json:"name"`
    Assets []Assets `json:"assets"`
}

type Assets struct {
    Name               string `json:"name"`
    BrowserDownloadUrl string `json:"browser_download_url"`
}

var client = http.DefaultClient

func main() {
    https_proxy := os.Getenv("https_proxy")
    if https_proxy != "" {
        proxy, _ := url.Parse(https_proxy)
        tr := &http.Transport{
            Proxy:           http.ProxyURL(proxy),
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
        client = &http.Client{
            Transport: tr,
            Timeout:   time.Second * 15, //超时时间
        }
    }

    url := "https://api.github.com/repos/MetaCubeX/Clash.Meta/releases"

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("content-type", "application/json")

    res, _ := client.Do(req)

    defer res.Body.Close()
    body, _ := io.ReadAll(res.Body)
    if res.StatusCode != 200 {
        fmt.Println("获取meta最新版本失败")
        return
    }
    var data []Release
    err := json.Unmarshal(body, &data)
    if err != nil {
        panic(err)
    }
    compile := regexp.MustCompile("Clash.Meta-windows-amd64-alpha-.+.zip")
    downloadUrl := ""
    name := ""
    for _, asset := range data[0].Assets {
        //fmt.Println(asset.Name)
        if compile.MatchString(asset.Name) {
            fmt.Println(asset.BrowserDownloadUrl)
            downloadUrl = asset.BrowserDownloadUrl
            name = asset.Name
            break
        }
    }
    if downloadUrl == "" {
        fmt.Println("未发现最新的alpha版本meta内核")
        return
    }
    //download(name, downlandUrl)
    spin := spinner.New(spinner.WithShape(components.Dot))
    spin.Display(func(spinner *spinner.Spinner) {
        spin.Refreshf("发现最新的alpha版本meta内核， 开始下载...")
        spinner.Refreshf("%s 下载中", name)
        group := downloadWithProgress(downloadUrl)
        group.Display()
        spin.Finish(name, " 下载完成")
    })

}

func download(name string, downloadUrl string) {

    req, _ := http.NewRequest("GET", downloadUrl, nil)
    resp, _ := client.Do(req)
    defer resp.Body.Close()

    f, _ := os.OpenFile("meta-alpha.zip", os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()

    //bar := progressbar.DefaultBytes(
    //   resp.ContentLength,
    //   "downloading",
    //)

    bar := progressbar.NewOptions(
        int(resp.ContentLength),
        progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
        progressbar.OptionEnableColorCodes(true),
        progressbar.OptionShowBytes(true),
        progressbar.OptionShowCount(),
        progressbar.OptionSetWidth(15),
        progressbar.OptionSetDescription("Downloading Clash.Meta"),
        progressbar.OptionSetTheme(progressbar.Theme{
            Saucer:        "[red]=[reset]",
            SaucerHead:    "[red]>[reset]",
            SaucerPadding: " ",
            BarStart:      "[",
            BarEnd:        "]",
        }))
    _, _ = io.Copy(io.MultiWriter(f, bar), resp.Body)
}

func downloadWithProgress(url string) *progress.Group {
    group := progress.NewGroupWithCount(1).
        AppendRunner(func(pro *components.Progress) func() {
            resp, err := http.Get(url)
            if err != nil {
                pro.Println("get error", err)
                resp.Body.Close()
                return func() {}
            }
            pro.WithTotal(resp.ContentLength)
            return func() {
                defer resp.Body.Close()

                dest, err := os.OpenFile(path.Base(url), os.O_CREATE|os.O_WRONLY, 0o777)
                defer dest.Close()
                if err != nil {
                    pro.Println("dest open error", err)
                    return
                }
                _, err = progress.StartTransfer(resp.Body, dest, pro)
                if err != nil {
                    pro.Println("trans error", err)
                }

            }
        })
    return group
}
