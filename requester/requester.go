// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package requester provides commands to run load tests and display results.
package requester

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	//"net/http/httptrace"
	"net/url"
	"os"
	"sync"
	"time"


	//"fmt"

)

// Max size of the buffer of result channel.
const maxResult = 1000000
const maxIdleConn = 500

type result struct {
	err           error
	statusCode    int
	duration      time.Duration
	connDuration  time.Duration // connection setup(DNS lookup + Dial up) duration
	dnsDuration   time.Duration // dns lookup duration
	reqDuration   time.Duration // request "write" duration
	resDuration   time.Duration // response "read" duration
	delayDuration time.Duration // delay between response and request
	contentLength int64
}

type Work struct {
	// Request is the request to be made.
	// request 是在hey.go中构造的
	Request *http.Request

	// requestbody 最初是在命令行里读取的
	RequestBody []byte

	// N is the total number of requests to make.
	N int

	// C is the concurrency level, the number of concurrent workers to run.
	C int

	// H2 is an option to make HTTP/2 requests
	H2 bool

	// Timeout in seconds.
	Timeout int

	// Qps is the rate limit in queries per second.
	QPS float64

	// DisableCompression is an option to disable compression in response
	DisableCompression bool

	// DisableKeepAlives is an option to prevents re-use of TCP connections between different HTTP requests
	DisableKeepAlives bool

	// DisableRedirects is an option to prevent the following of HTTP redirects
	DisableRedirects bool

	// Output represents the output type. If "csv" is provided, the
	// output will be dumped as a csv stream.
	Output string

	// ProxyAddr is the address of HTTP proxy server in the format on "host:port".
	// Optional.
	ProxyAddr *url.URL

	// Writer is where results will be written. If nil, results are written to stdout.
	Writer io.Writer

	initOnce sync.Once
	results  chan *result
	stopCh   chan struct{}
	start    time.Duration

	report *report
}

func (b *Work) writer() io.Writer {
	if b.Writer == nil {
		return os.Stdout
	}
	return b.Writer
}

// Init initializes internal data-structures
func (b *Work) Init() {
	b.initOnce.Do(func() {
		//10个人 开10000个result?
		b.results = make(chan *result, min(b.C*1000, maxResult))
		//stopCh 优雅的停止client需要的signal
		b.stopCh = make(chan struct{}, b.C)
	})
}

// Run makes all the requests, prints the summary. It blocks until
// all work is done.
func (b *Work) Run() {
	b.Init()
	b.start = now()
	//初始化report系统
	b.report = newReport(b.writer(), b.results, b.Output, b.N)
	//创建report线程, it polls the result channel until it is closed.
	go func() {
		runReporter(b.report)
	}()
	b.runWorkers()
	b.Finish()
}


//主程序 hey接受到终端的中断信号以后 调用stop()
func (b *Work) Stop() {
	// Send stop signal so that workers can stop gracefully.
	for i := 0; i < b.C; i++ {
		b.stopCh <- struct{}{}
	}
}

func (b *Work) Finish() {
	close(b.results)
	total := now() - b.start
	// Wait until the reporter is done.
	<-b.report.done

	//finalize里面才是计算，最后去话，时间不计算在这里面
	b.report.finalize(total)
}


//具体发出request请求的方法
func (b *Work) makeRequest(c *http.Client) {
	s := now()
	var size int64
	var code int
	//var dnsStart, connStart, resStart, reqStart, delayStart time.Duration
	//var dnsDuration, connDuration, resDuration, reqDuration, delayDuration time.Duration

	// clone() 存在的主要原因是允许 Body 对象的多次使用,每次复用同一个数据结构, 注释掉 感觉对速度影响不大
	// 我自己写的程序可能慢就慢在每次都要new一个request
	req := cloneRequest(b.Request, b.RequestBody)
/*
	//添加跟踪器 类似于回调函数，显示在外层的 details里面
	trace := &httptrace.ClientTrace{
		// 计算DNS lookup durantion ，dns解析所需时间 dnsDuration = DNS-lookup
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsDuration = now() - dnsStart
		},

		//得到连接所需要的duration, connDuration = DNS+dialup = connection setup(DNS lookup + Dial up) duration
		GetConn: func(h string) {
			connStart = now()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			if !connInfo.Reused {
				connDuration = now() - connStart
			}
			reqStart = now()
		},

		//reqDuration(request duration) = resp write
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			reqDuration = now() - reqStart
			delayStart = now()
		},

		//delayDuration = resp wait
		GotFirstResponseByte: func() {
			delayDuration = now() - delayStart
			resStart = now()
		},
	}

	//把trace 写入
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
*/
	//此处真正发出request， client.DO(request) 系统方法
	resp, err := c.Do(req)

	//正常返回
	if err == nil {
		//返回-1 长度未知, 如果是helloworld 会返回实际字符串长度
		size = resp.ContentLength
		//返回码 200， 每次都是要确认200？
		code = resp.StatusCode

		//这段属于自己加戏，源代码中没有，返回byte类型，用string（打印）
		//body, err := ioutil.ReadAll(resp.Body)
		//if err == nil {
		//	fmt.Print("Server:",string(body),"\n")
		//}



		//如果没有下面操作 qps会变小 速度变慢
		//如果body既没有被完全读取，也没有被关闭，那么这次http事务就没有完成，除非连接因超时终止了，否则相关资源无法被回收。
		//抛弃没有读完的body.
		io.Copy(ioutil.Discard, resp.Body)
		//回收资源 连接释放.
		resp.Body.Close()
	}


	t := now()
	//最后一个duration, resDuration(response duration)=  resp read
	//resDuration = t - resStart
	//一个request所需要的时间。
	finish := t - s

	//把结果写入result
	b.results <- &result{
		statusCode:    code,
		duration:      finish,
		err:           err,
		contentLength: size,

		// 这些detail的参数暂时用不到 注释了
		connDuration:  time.Second,//connDuration,
		dnsDuration:   time.Second,//dnsDuration,
		reqDuration:   time.Second,//reqDuration,
		resDuration:   time.Second,//resDuration,
		delayDuration: time.Second,//delayDuration,
	}
}

func (b *Work) runWorker(client *http.Client, n int) {
	//创建一个time.Time类型的channel ，节流阀？
	//这个功能现在不需要 注释了
	//var throttle <-chan time.Time
	// 默认没有速率限制， 你测qps为什么还要限制qps？？
	//if b.QPS > 0 {
		//time.Tick 每隔一段时间向throttle channel里写一个时间数据
		//时间长度的格式 一定要 time.Duration(*) * time.(*)
		//1秒可以是 time.Duration(1) * time.Second 注意；这里的1可以是浮点数10/9(但是不能写成1.1，更不能小于1会报错)
		//1秒= time.Duration(1000) * time.Millisecond = time.Duration(1e6) * time.Microsecond = time.Second
		//这里除以QPS  就是每个request的时间间隔.
		//当然这个功能现在不需要 注释了
		//throttle = time.Tick(time.Duration(1e6/(b.QPS)) * time.Microsecond)
	//}


	//URL 重定向，也称为 URL 转发，是一种当实际资源，如单个页面、表单或者整个 Web 应用被迁移到新的 URL 下的时候，保持（原有）链接可用的技术。
	//暂时不用 注释
	//if b.DisableRedirects {
	//	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
	//		return http.ErrUseLastResponse
	//	}
	//}

	for i := 0; i < n; i++ {
		// Check if application is stopped. Do not send into a closed channel.

		//select语法：
		//每个case都必须是一个通信

		//所有channel表达式都会被求值 （case getChan(0) <- getNumber(2): 重要！：求值顺序：自上而下、从左到右 ）
		//所有被发送的表达式都会被求值

		//如果任意某个通信可以进行，它就执行；其他被忽略。
		//如果有多个case都可以运行，Select会随机公平地选出一个执行。其他不会执行。
		//否则：
		//如果有default子句，则执行该语句。
		//如果没有default字句，select将阻塞，直到某个通信可以运行；Go不会重新对channel或值进行求值。
		select {
		// stop()里面会发送这个信号
		case <-b.stopCh:

			return
		default:
			//if b.QPS > 0 {
				//根据qps设定每秒钟执行几次 实际中我们用不到
			//	<-throttle
			//}
			b.makeRequest(client)
		}
	}
}

func (b *Work) runWorkers() {
	var wg sync.WaitGroup
	//添加任务数 （client的任务数）
	wg.Add(b.C)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			//不再对服务端的证书进行校验
			InsecureSkipVerify: true,
			//ServerName : localhost:9090
			ServerName:         b.Request.Host,
		},
		//每个主机下的最大闲置连接数目 maxIdleConn = 500 最多开500个clicent线程
		MaxIdleConnsPerHost: min(b.C, maxIdleConn),
		// 这个暂时不用 注释了
		//禁止压缩头 默认是false 就是可以允许压缩
		//DisableCompression:  b.DisableCompression,
		// 这个暂时不用 注释了
		//http中的吧
		//DisableKeepAlives:   b.DisableKeepAlives,
		// 这个暂时不用 注释了
		// Proxy指定一个对给定请求返回代理的函数。如果该函数返回了非nil的错误值，请求的执行就会中断并返回该错误。
		// 如果Proxy为nil或返回nil的*URL置，将不使用代理。
		//Proxy:               http.ProxyURL(b.ProxyAddr),
	}
	if b.H2 {
		//h2这段删掉了
	} else {
		// 注意！！：这个我自己注释掉了 目前没有影响。
		//当Go的客户端使用HTTPS的时候会默认使用 HTTP2 协议。这里并没有针对HTTP2协议改变包的接口。
		// 如果客户端需要禁止 HTTP2 协议，可以通过将 设置为非nil的空map实现。
		//tr.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}

	// Transport用的就是上面的tr
	// timeout是命令行读入 默认20s
	client := &http.Client{Transport: tr, Timeout: time.Duration(b.Timeout) * time.Second}

	// 每个client开一个goroutine， go的数量和client数量一致
	// Ignore the case where b.N % b.C != 0.
	for i := 0; i < b.C; i++ {
		go func() {

			//每个client内部执行的request数量= 总数/client数量
			b.runWorker(client, b.N/b.C)
			//一个client完成后，计数减1
			wg.Done()
		}()
	}

	//阻塞 直到所有任务结束
	wg.Wait()
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
// clone() 存在的主要原因是允许 Body 对象的多次使用
func cloneRequest(r *http.Request, body []byte) *http.Request {
	// struct 浅copy
	// 每个请求都要一个new一个新的request
	r2 := new(http.Request)
	*r2 = *r

	//make也是用于内存分配的，但是和new不同，它只用于chan、map以及切片的内存创建.
	//而且它返回的类型就是这三个类型本身，而不是他们的指针类型，因为这三种类型就是引用类型，所以就没有必要返回他们的指针了。

	// struct 深copy
	r2.Header = make(http.Header, len(r.Header))

	// append方法例子:
	//x := []int {1,2,3}
	//y := []int {4,5,6}
	//fmt.Println(append(x,4,5,6))
	//fmt.Println(append(x,y...));
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
		//Content-Type:[text/html] User-Agent:[hey/0.0.1]
		//fmt.Print(k,":",r2.Header[k],"\n")
	}

	// 在makeRequest body都会被清空
	// body都是空， 没有进去过
	if len(body) > 0 {
		// NopCloser 将 r 包装为一个 ReadCloser 类型，但 Close 方法不做任何事情。
		r2.Body = ioutil.NopCloser(bytes.NewReader(body))
		//fmt.Print("Server2222:",string(body),"/n")

	}
	return r2
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
