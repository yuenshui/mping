package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const URL_INDEX = "https://ping.aizhan.com/"
const URL_HOST = "https://ping.aizhan.com"
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36"

var cookie = make(map[string]string)
var domain = ""
var IPs sync.Map

type IPInfo struct {
	min string
	max string
	avg string
	IP  string
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("mping github.com")
		return
	}

	domain = os.Args[1]
	fmt.Println("domain:", domain)
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(URL_INDEX)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		return
	}
	defer resp.Body.Close()

	header, ok := resp.Header["Set-Cookie"]

	spaceRe, _ := regexp.Compile(`;\s*`)
	wRe, _ := regexp.Compile(`\s*=\s*`)

	if ok {
		for _, line := range header {
			ss := spaceRe.Split(line, -1)
			if len(ss) < 2 {
				continue
			}
			vv := wRe.Split(ss[0], -1)
			if len(vv) < 2 {
				continue
			}
			cookie[vv[0]] = vv[1]
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("body error ", err.Error())
		return
	}
	searchForm, dataForm, err := parseFormConfig(string(body))
	csrf, _ := (*searchForm)["_csrf"]
	path, _ := (*searchForm)["url"]

	listString, err := aizhanHTTPPOST(&csrf, &path, nil)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// fmt.Println(listString)
	list, err := parseList(listString)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	path = (*dataForm)["url"]

	fmt.Println("需要等待几分钟搜集IP，并分析创建连接相应时间。")
	fmt.Println("If the previous line is garbled, it is not an error, but your system does not support Chinese fonts.")
	fmt.Println("You need to wait a few minutes to collect the IP and analyze the corresponding time to create the connection.")
	wg := &sync.WaitGroup{}
	for _, nodeId := range list {
		wg.Add(1)
		loadIp(&csrf, &path, &nodeId, wg)
	}
	wg.Wait()
}

func parseFormConfig(body string) (searchFor *map[string]string, dataFor *map[string]string, err error) {
	lineRe, _ := regexp.Compile(`[\r\n]{1,2}`)
	bodyArr := lineRe.Split(string(body), -1)
	searchFormStart := 0
	dataFormStart := 0
	for index, line := range bodyArr {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "function search_form") {
			searchFormStart = index
		}
		if strings.Contains(line, "function getData") {
			dataFormStart = index
		}
	}

	searchForm, searchFormErr := parseForm(bodyArr, searchFormStart)
	dataForm, dataFormStartErr := parseForm(bodyArr, dataFormStart)
	if searchFormErr != nil {
		err = searchFormErr
	} else if dataFormStartErr != nil {
		err = dataFormStartErr
	} else {
		err = nil
	}
	return searchForm, dataForm, err
}

func parseForm(body []string, index int) (*map[string]string, error) {
	kvRe := regexp.MustCompile(`^(\w+)\s*:\s*"?(.*)"?,`)
	dataRe := regexp.MustCompile(`_csrf:'(.*)'`)

	var err error
	form := make(map[string]string)
	if index > 0 {
		for i := 0; i < 25; i++ {
			line := body[index+i]
			constant := strings.TrimSpace(line)
			mc := kvRe.FindSubmatch([]byte(constant))
			if len(mc) > 0 {
				form[string(mc[1])] = strings.TrimRight(string(mc[2]), "\"")
			}
		}

	} else {
		err = errors.New("index must eq 0")
	}
	data, ok := form["data"]
	if ok {
		dataRs := dataRe.FindSubmatch([]byte(data))
		if len(dataRs) > 0 {
			form["_csrf"] = string(dataRs[1])
		}
	}

	return &form, err
}

func parseList(txt *string) ([]string, error) {
	type nodeInfo struct {
		NodeId   string `json:"node_id"`
		NodeName string `json:"node_name"`
	}
	type listFomat struct {
		Node []nodeInfo `json:"node"`
	}
	var data listFomat
	err := json.Unmarshal([]byte(*txt), &data)
	if err != nil {
		return nil, errors.New("parseList parse json error")
	}
	nodeIds := []string{}
	for _, node := range data.Node {
		nodeIds = append(nodeIds, node.NodeId)
	}
	return nodeIds, nil
}

func cookieBuild() string {
	var strArr []string
	for k, v := range cookie {
		strArr = append(strArr, k+"="+v)
	}
	return strings.Join(strArr, ";")
}

func aizhanHTTPPOST(csrf, path, nodeId *string) (*string, error) {
	urlfull := URL_HOST + *path
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(25 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*20)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
		},
	}

	form := url.Values{}
	form.Set("type", "http")
	form.Set("domain", domain)
	form.Set("_csrf", *csrf)
	if nodeId != nil && *nodeId != "" {
		form.Set("node_id", *nodeId)
	}

	b := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", urlfull, b)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Cookie", cookieBuild())
	req.Header.Set("Referer", URL_INDEX)
	req.Header.Set("Origin", URL_HOST)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(0)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	bodyString := string(body)
	return &bodyString, err
}

func parseNode(txt *string) (*string, error) {
	type NodeInfo struct {
		Ip     string `json:"Ip"`
		Status string `json:"status"`
	}
	var data NodeInfo
	err := json.Unmarshal([]byte(*txt), &data)
	if err != nil {
		return nil, errors.New("parseNode parse json error")
	}
	return &data.Ip, nil
}

func loadIp(csrf, path, nodeId *string, wg *sync.WaitGroup) {
	nodeString, err := aizhanHTTPPOST(csrf, path, nodeId)
	if err != nil {
		wg.Done()
		return
	}

	ip, err := parseNode(nodeString)
	if err != nil || ip == nil || *ip == "" {
		wg.Done()
		return
	}

	_, ok := IPs.Load(*ip)
	if ok {
		wg.Done()
		return
	}
	IPs.Store(*ip, true)
	res := []byte{}

	if runtime.GOOS == "windows" {
		res, err = exec.Command("ping", *ip).Output()
	} else {
		res, err = exec.Command("ping", "-c", "3", *ip).Output()
	}

	if err != nil {
		wg.Done()
		return
	}
	ipInfo := parseIpSources(res)

	fmt.Println(ipInfo.IP, "min:"+ipInfo.min, "max:"+ipInfo.max, "avg:"+ipInfo.avg)
	wg.Done()
}

func parseIpSources(txt []byte) *IPInfo {
	var parsePing *regexp.Regexp
	if runtime.GOOS == "windows" {
		parsePing = regexp.MustCompile(`([\.\d]+)ms[^\d]+([\.\d]+)ms[^\d]+([\.\d]+)ms`)
	} else {
		parsePing = regexp.MustCompile(`(\w+)/(\w+)/(\w+)/(\w+)?\s*=\s*([\.\d]+)/([\.\d]+)/([\.\d]+)/([\.\d]+)?`)
	}

	parseIP := regexp.MustCompile(`(?i)(PING)\s+([^\s]+)\s*([^\d]?([\d\.]{7,15})[^\d]?)?`)

	rs := parsePing.FindSubmatch(txt)
	info := &IPInfo{}
	if len(rs) == 9 {
		info.min = string(rs[5])
		info.max = string(rs[7])
		info.avg = string(rs[6])
	} else if len(rs) == 4 {
		info.min = string(rs[1])
		info.max = string(rs[2])
		info.avg = string(rs[3])
	} else {
		info.min = "-"
		info.max = "-"
		info.avg = "-"
	}

	rs = parseIP.FindSubmatch(txt)
	if len(rs) == 5 {
		if len(rs[4]) > 0 {
			info.IP = string(rs[4])
		} else {
			info.IP = string(rs[2])
		}
	} else {
		info.IP = "-"
	}

	return info
}
