package main

import (
	"bytes"
	"fmt"
	"github.com/gocolly/colly/extensions"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)
import "github.com/gocolly/colly"

// PathIsExists 判断所给路径文件/文件夹是否存在
func PathIsExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// IsDir 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}

func DirExistsAndCreate(dir string) {
	//判断是否存在存储目录 没有就创建
	if !PathIsExists(dir) {
		os.Mkdir(dir, os.ModePerm)
	}
}

func main() {
	/*创建字典 */
	var valueMap map[string]string
	valueMap = make(map[string]string)
	valueMap["storePath"] = "telegraphImgs"
	mutex := &sync.Mutex{}
	imgCounter := 0

	DirExistsAndCreate("./" + valueMap["storePath"])
	// 访问地址
	//postUrl := "https://telegra.ph/%E7%A5%9E%E6%A5%BD%E5%9D%82%E7%9C%9F%E5%86%AC---21%E5%B9%B409%E6%9C%88%E5%BE%AE%E5%8D%9A%E8%AE%A2%E9%98%85-80P-182MB-05-15"
	var postUrl string
	fmt.Println("请输入浏览器telegraph页面url地址：")
	fmt.Scanln(&postUrl)
	const Socks5ProxyUrl = "socks5://127.0.0.1:7892"

	const baseUrl = "https://telegra.ph"

	timeStart := time.Now()

	// 实例化默认收集器
	c := colly.NewCollector(func(collector *colly.Collector) {
		// 表示抓取时异步的
		collector.Async = true
		// 模拟浏览器
		extensions.RandomUserAgent(collector)
		// 仅访问域

		collector.AllowedDomains = []string{"telegra.ph"}
		//设置代理
		collector.SetProxy(Socks5ProxyUrl)
		//设置请求超时时间
		collector.SetRequestTimeout(120 * time.Second)
	})

	// 限制采集规则
	/*
		在Colly里面非常方便控制并发度，只抓取符合某个(些)规则的URLS
		colly.LimitRule{DomainGlob: "*.douban.*", Parallelism: 5}，表示限制只抓取域名是douban(域名后缀和二级域名不限制)的地址，当然还支持正则匹配某些符合的 URLS

		Limit方法中也限制了并发是5。为什么要控制并发度呢？因为抓取的瓶颈往往来自对方网站的抓取频率的限制，如果在一段时间内达到某个抓取频率很容易被封，所以我们要控制抓取的频率。
		另外为了不给对方网站带来额外的压力和资源消耗，也应该控制你的抓取机制。
	*/
	// err := c.session.Limit(&colly.LimitRule{DomainGlob: "*.quotes.*", Parallelism: 5})
	// if err != nil {
	// 	fmt.Println(err)
	// }

	//图片下载器
	imageCollector := c.Clone()
	//imageCollector.Limit(&colly.LimitRule{
	//	RandomDelay: 2 * time.Second,
	//	Parallelism: 10,
	//})

	//图片请求
	imageCollector.OnRequest(func(request *colly.Request) {
		requestURL := request.URL.String()
		splitURL := strings.Split(requestURL, "/")
		fileName := splitURL[len(splitURL)-1]
		imgPathInServer := "/file/" + fileName
		//println(imgPathInServer)
		request.Headers.Set("authority", baseUrl)
		request.Headers.Set("method", "GET")
		request.Headers.Set("path", imgPathInServer)
		request.Headers.Set("accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		request.Headers.Set("accept-encoding", "gzip, deflate, br")
		request.Headers.Set("accept-language", "zh,en;q=0.9,zh-TW;q=0.8,zh-CN;q=0.7,ja;q=0.6")
		request.Headers.Set("cache-control", "no-cache")
		request.Headers.Set("pragma", "no-cache")
		request.Headers.Set("referrer", postUrl)
		request.Headers.Set("sec-ch-ua", " Not A;Brand\";v=\"99\", \"Chromium\";v=\"102\", \"Google Chrome\";v=\"102")
		request.Headers.Set("sec-ch-ua-mobile", "?0")
		request.Headers.Set("sec-ch-ua-platform", "macOS")
		request.Headers.Set("sec-fetch-dest", "image")
		request.Headers.Set("sec-fetch-mode", "no-cors")
		request.Headers.Set("sec-fetch-site", "same-origin")
		request.Headers.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.61 Safari/537.36")
		request.Headers.Set("scheme", "https")

	})

	//图片响应
	imageCollector.OnResponse(func(response *colly.Response) {
		imgPathInServer := response.Request.Headers.Get("path")
		splitURL := strings.Split(imgPathInServer, "/")
		fileName := splitURL[len(splitURL)-1]

		fileStorePath := "./" + filepath.Join(valueMap["storePath"], valueMap["title"], fileName)
		file, err := os.Create(fileStorePath)
		if err != nil {
			panic(err)
		}
		io.Copy(file, bytes.NewReader(response.Body))

		mutex.Lock()
		imgCounter += 1
		mutex.Unlock()
		fmt.Printf("第%d张图片: %s下载完成......\r\n", imgCounter, imgPathInServer)

	})

	//图片下载出错
	imageCollector.OnError(func(response *colly.Response, err error) {
		s := response.Request.URL.String()
		println(err.Error())
		fmt.Printf("图片链接: %s 下载出错 %v......", s, err)
	})
	//获取标题
	c.OnHTML("link[rel=\"canonical\"]", func(element *colly.HTMLElement) {
		title := element.Attr("href")
		Splitter := strings.Split(title, "/")
		titleName := Splitter[len(Splitter)-1]
		//println(title)
		valueMap["title"] = titleName
		storeEachItemDir := "./" + valueMap["storePath"] + "/" + titleName
		DirExistsAndCreate(storeEachItemDir)
	})

	c.OnHTML("img", func(e *colly.HTMLElement) {
		// 获取属性值
		link := e.Attr("src")
		//fmt.Printf("Link found: %q -> %s\n", e.Text, link)
		var imageUrl string = baseUrl + link
		//println(imageUrl)
		splitURL := strings.Split(link, "/")
		fileName := splitURL[len(splitURL)-1]
		imageStorePath := "./" + filepath.Join(valueMap["storePath"], valueMap["title"], fileName)
		//排除重复下载的
		if !PathIsExists(imageStorePath) {
			//请求图片
			imageCollector.Visit(imageUrl)
		} else {
			fmt.Printf("图片：%s 已下载，跳过......\r\n", fileName)
		}

	})

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	// 结束
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	// 开始爬取 url
	err2 := c.Visit(postUrl)
	fmt.Printf("开始爬取链接：%s\n", postUrl)
	fmt.Printf("开始解析网页,请稍后......\n")

	if err2 != nil {
		fmt.Printf("出现错误：%s\n", err2)
	}

	// 采集等待结束
	c.Wait()
	imageCollector.Wait()

	fmt.Printf("共下载 %d 张图片, 耗时: %v s", imgCounter, time.Since(timeStart).Seconds())
}
