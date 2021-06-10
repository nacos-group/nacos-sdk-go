/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package naming_client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/util"
)

type PushReceiver struct {
	port        int
	host        string
	hostReactor *HostReactor
}

type PushData struct {
	PushType    string `json:"type"`
	Data        string `json:"data"`
	LastRefTime int64  `json:"lastRefTime"`
}

var (
	GZIP_MAGIC = []byte("\x1F\x8B")
)

func NewPushReceiver(hostReactor *HostReactor) *PushReceiver {
	pr := PushReceiver{
		hostReactor: hostReactor,
	}
	pr.startServer()
	return &pr
}

func (us *PushReceiver) tryListen() (*net.UDPConn, bool) {
	addr, err := net.ResolveUDPAddr("udp", us.host+":"+strconv.Itoa(us.port))
	if err != nil {
		logger.Errorf("can't resolve address,err: %+v", err)
		return nil, false
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logger.Errorf("error listening %s:%d,err:%+v", us.host, us.port, err)
		return nil, false
	}

	return conn, true
}

const (
	PushReceiverListenPortEnvKey     = "PUSH_RECEIVER_LISTEN_PORT"
	PushReceiverListenRetryTimes     = "PUSH_RECEIVER_RETRY_TIMES"
	PushReceiverExitCodeAfterRetried = "PUSH_RECEIVER_EXIT_CODE_AFTER_RETRIED"

	defaultRetryTimes = 3
)

var (
	portPattern      = regexp.MustCompile(`^(?P<port>\d{1,5})$`)
	portRangePattern = regexp.MustCompile(`^(?P<begin>\d{1,5})[-~](?P<end>\d{1,5})$`)

	portIndex  = 1
	beginIndex = 1
	endIndex   = 2
)

func validPort(port int) bool {
	return port > 1024 && port < 65535
}

func getListenPort() int {
	raw := os.Getenv(PushReceiverListenPortEnvKey)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if matches := portPattern.FindStringSubmatch(raw); matches != nil {
		if port, err := strconv.Atoi(matches[portIndex]); err != nil {
			logger.Warnf("parse push receiver listening port failed, env: %s, err: %s", raw, err)
		} else if !validPort(port) {
			logger.Warnf("specified push receiver listening port invalid, 1024 ~ 65535, env: %s", raw)
		} else {
			logger.Infof("use specified port for push receiver listening, port: %d", port)
			return port
		}
	} else if matches = portRangePattern.FindStringSubmatch(raw); matches != nil {
		begin, err := strconv.Atoi(matches[beginIndex])
		if err != nil {
			logger.Warnf("parse push receiver listening port range begin failed, env: %s, err: %s", raw, err)
		}
		end, err := strconv.Atoi(matches[endIndex])
		if err != nil {
			logger.Warnf("parse push receiver listening port range end failed, env: %s, err: %s", raw, err)
		}
		if validPort(begin) && validPort(end) {
			if begin > end {
				t := end
				end = begin
				begin = t
			}
			port := r.Intn(end-begin) + begin
			logger.Infof("use random port from specified range for push receiver listening, port: %d", port)
			return port
		}
		logger.Warnf("specified push receiver listening port range invalid, 1024 ~ 65535, env: %s", raw)
	}
	port := r.Intn(1000) + 54951
	logger.Infof("use random port for push receiver listening, port: %d", port)
	return port
}

func getRetryTimes() int {
	raw := os.Getenv(PushReceiverListenRetryTimes)
	if raw == "" {
		logger.Infof("use default retry times for push receiver listening, times: %d", defaultRetryTimes)
		return defaultRetryTimes
	}
	times, err := strconv.Atoi(raw)
	if err != nil {
		logger.Warnf("parse retry times for push receiver listening failed, use default, env: %s", raw)
		return defaultRetryTimes
	}
	logger.Infof("use specified retry times for push receiver listening, times: %d", times)
	return times
}

func shouldExit() (bool, int) {
	raw := os.Getenv(PushReceiverExitCodeAfterRetried)
	if raw == "" {
		return false, 0
	}
	code, err := strconv.Atoi(raw)
	if err != nil {
		logger.Warnf("parse exit code of push receiver listen after retried failed, not to exit, env: %s, err: %s", raw, err)
		return false, 0
	}
	logger.Infof("use specified exit code if push receiver listen failed after retried, code: %d", code)
	return true, code
}

func (us *PushReceiver) getConn() *net.UDPConn {
	var conn *net.UDPConn
	retryTimes := getRetryTimes()
	for i := 0; i < retryTimes; i++ {
		us.port = getListenPort()
		conn1, ok := us.tryListen()

		if ok {
			conn = conn1
			logger.Infof("udp server start, port: %d", us.port)
			return conn
		}

		if !ok && i == retryTimes-1 {
			logger.Errorf("failed to start udp server after trying 3 times.")
			if should, code := shouldExit(); should {
				os.Exit(code)
			}
		}
	}
	return nil
}

func (us *PushReceiver) startServer() {
	conn := us.getConn()
	go func() {
		defer func() {
			if conn != nil {
				conn.Close()
			}
		}()
		for {
			us.handleClient(conn)
		}
	}()
}

func (us *PushReceiver) handleClient(conn *net.UDPConn) {

	if conn == nil {
		time.Sleep(time.Second * 5)
		conn = us.getConn()
		if conn == nil {
			return
		}
	}

	data := make([]byte, 4024)
	n, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		logger.Errorf("failed to read UDP msg because of %+v", err)
		return
	}

	s := TryDecompressData(data[:n])
	logger.Info("receive push: "+s+" from: ", remoteAddr)

	var pushData PushData
	err1 := json.Unmarshal([]byte(s), &pushData)
	if err1 != nil {
		logger.Infof("failed to process push data.err:%+v", err1)
		return
	}
	ack := make(map[string]string)

	if pushData.PushType == "dom" || pushData.PushType == "service" {
		us.hostReactor.ProcessServiceJson(pushData.Data)

		ack["type"] = "push-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""

	} else if pushData.PushType == "dump" {
		ack["type"] = "dump-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = util.ToJsonString(us.hostReactor.serviceInfoMap)
	} else {
		ack["type"] = "unknow-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	}

	bs, _ := json.Marshal(ack)
	c, err := conn.WriteToUDP(bs, remoteAddr)
	if err != nil {
		logger.Errorf("WriteToUDP failed,return:%d,err:%+v", c, err)
	}
}

func TryDecompressData(data []byte) string {

	if !IsGzipFile(data) {
		return string(data)
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))

	if err != nil {
		logger.Errorf("failed to decompress gzip data,err:%+v", err)
		return ""
	}

	defer reader.Close()
	bs, err := ioutil.ReadAll(reader)

	if err != nil {
		logger.Errorf("failed to decompress gzip data,err:%+v", err)
		return ""
	}

	return string(bs)
}

func IsGzipFile(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	return bytes.HasPrefix(data, GZIP_MAGIC)
}
