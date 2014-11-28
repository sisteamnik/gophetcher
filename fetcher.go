package gophetcher

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

type (
	Fetcher struct {
		urls    chan string
		wg      sync.WaitGroup
		stop    chan interface{}
		workers int
		hl      func(*FetchResponse)
		Client  *http.Client
	}

	FetchResponse struct {
		Header string
		Body   []byte

		//query duration in nanoseconds
		Duration  int64
		TargetUrl string

		//non-empty if redirect
		FinalUrl     string
		ResponseCode int
		Date         string
		Ip           string
	}

	FetchResponses struct {
		responses []FetchResponse
	}
)

func NewFetcher(handler func(*FetchResponse)) *Fetcher {
	f := new(Fetcher)
	f.urls = make(chan string)
	f.workers = 20
	f.hl = handler
	return f
}

func (f *Fetcher) Start() {
	for i := 0; i < f.workers; i++ {
		go func() {
			for {
				select {
				case v := <-f.urls:
					r, _ := f.fetch(v)
					f.wg.Done()
					f.hl(r)
				}
			}
		}()
	}
}

func (f *Fetcher) Wait() {
	f.wg.Wait()
}

func (f *Fetcher) Send(url ...string) FetchResponses {
	for _, v := range url {
		f.wg.Add(1)
		f.urls <- v
	}
	return FetchResponses{}
}

func (f *Fetcher) fetch(url string) (fr *FetchResponse, err error) {
	fr = new(FetchResponse)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	ts := time.Now()
	resp, err := client.Do(req)
	fr.Duration = time.Now().Sub(ts).Nanoseconds()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fr.Body, err = ioutil.ReadAll(resp.Body)
	fr.Ip = ip(req.Host)
	fr.TargetUrl = url
	finalurl := req.URL.String()
	if finalurl != url {
		fr.FinalUrl = finalurl
	}
	fr.Date = time.Now().Format(time.RFC3339)
	fr.Header = header(resp)
	fr.ResponseCode = resp.StatusCode
	return
}

func ip(host string) string {
	u, _ := net.LookupIP(host)
	if len(u) > 0 {
		return u[0].String()
	}
	return ""
}

func header(res *http.Response) string {
	buf := new(bytes.Buffer)
	res.Header.Write(buf)
	return buf.String()
}
