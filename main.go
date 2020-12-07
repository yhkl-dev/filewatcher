package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v1"
)

const configFileName = "./conf/monitor.yml"

type conf struct {
	DingDingURL    string `yaml:"dingdingurl"`
	DingDingSecret string `yaml:"secretkey"`
	FileName       string `yaml:"filename"`
	MonitorString  string `yaml:"monitorString"`
}

func (c *conf) getConf() *conf {
	yamlFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

func processTask(line []byte, config *conf) {
	// 	os.Stdout.Write(line)
	//fmt.Println(string(line))
	re := regexp.MustCompile(config.MonitorString)
	result := re.Find(line)
	if matched, _ := regexp.MatchString(config.MonitorString, string(line)); matched {
		fmt.Println("matched", string(result))
		sendDingDingMessage(string(result), config.DingDingURL, config.DingDingSecret)
	}
}

func fileMonitoring(filePth string, hookfn func(line []byte, config *conf), config *conf) {
	f, err := os.Open(filePth)
	if err != nil {
		log.Fatalln(err)

	}
	defer f.Close()

	rd := bufio.NewReader(f)
	f.Seek(0, 2)
	for {
		line, err := rd.ReadBytes('\n')
		// 如果是文件末尾不返回
		if err == io.EOF {
			time.Sleep(500 * time.Millisecond)
			continue

		} else if err != nil {
			log.Fatalln(err)

		}
		go hookfn(line, config)
	}
}

func hmacSha256(stringToSign string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func sendDingDingMessage(content string, dingdingURL string, secretKey string) {

	timestamp := time.Now().UnixNano() / 1e6
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secretKey)
	sign := hmacSha256(stringToSign, secretKey)

	data := `{"msgtype":"markdown","markdown":{"title":"系统异常","text":"%s"},"at":{"atMobiles":[],"isAtAll":false}}`
	fmtDat := fmt.Sprintf(data, content)

	var jsonStr = []byte(fmtDat)
	buffer := bytes.NewBuffer(jsonStr)
	postURL := fmt.Sprintf("%s&timestamp=%d&sign=%s", dingdingURL, timestamp, sign)

	fmt.Println(postURL)
	request, err := http.NewRequest("POST", postURL, buffer)
	if err != nil {
		fmt.Println(err)
	}
	request.Header.Set("Content-Type", "application/json;charset=utf-8")
	client := http.Client{}
	resp, err := client.Do(request)

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func main() {
	info := conf{}
	config := info.getConf()
	fileMonitoring("./tmp/test.log", processTask, config)
}
