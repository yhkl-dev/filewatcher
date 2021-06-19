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
	"strings"
	"time"

	"gopkg.in/yaml.v1"
)

const configFileName = "./conf/monitor.yml"

type conf struct {
	DingDingURL    string `yaml:"dingdingurl"`
	DingDingSecret string `yaml:"secretkey"`
	FileName       string `yaml:"filename"`
	System         string `yaml:"system`
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

func processTask(system string, line []byte, config *conf) {
	re := regexp.MustCompile(config.MonitorString)
	result := re.Find(line)
	if matched, _ := regexp.MatchString(config.MonitorString, string(line)); matched {
		fmt.Println("matched", string(line))
		fmt.Println("matched", string(result))
		go sendDingDingMessage(system, strings.Replace(string(line), "\"", "'", -1), config.DingDingURL, config.DingDingSecret)
	}
}

func fileMonitoring(system, filePth string, hookfn func(system string, line []byte, config *conf), config *conf) {
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
		go hookfn(system, line, config)
	}
}

func hmacSha256(stringToSign string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func sendDingDingMessage(system, content string, dingdingURL string, secretKey string) {

	timestamp := time.Now().UnixNano() / 1e6
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secretKey)
	sign := hmacSha256(stringToSign, secretKey)

	// data := `{"msgtype":"markdown","markdown":{"title":"%s","text":"%s%s"},"at":{"atMobiles":[],"isAtAll":false}}`
	// headData := fmt.Sprintf("## %s\n", system)
	//fmtDat := fmt.Sprintf(data, system, headData, content)
	data := `{"msgtype":"markdown","markdown":{"title":"%s","text":"### system: %s \n\n%s"},"at":{"atMobiles":[],"isAtAll":false}}`
	fmtDat := fmt.Sprintf(data, system, system, content)

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
	fileMonitoring(config.System, config.FileName, processTask, config)
	// s := `2021-06-19 17:50:54.104 ERROR 21771 --- [-8082-exec-7089] c.j.r.c.e.ExceptionControllerAdvice :"" 用户未启用，请启用后再登录`
	// newS := strings.Replace(s, "\"", "'", -1)
	// fmt.Println(newS)
}
