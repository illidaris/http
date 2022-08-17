package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	nUrl "net/url"
	"os"
	"strings"
	"time"

	"github.com/illidaris/core"
	"github.com/illidaris/logger"
)

const (
	ProxyURLKey string = "_proxy_url"
)

var (
	DefaultClient = func(ctx context.Context) *http.Client {
		v := ctx.Value(ProxyURLKey)
		if v != nil {
			if url, ok := v.(string); ok {
				client := &http.Client{
					Transport: &http.Transport{
						Proxy: func(r *http.Request) (*nUrl.URL, error) {
							return nUrl.Parse(url)
						},
					},
					Timeout: time.Second * 60,
				}
				return client
			}
		}
		client := http.DefaultClient
		client.Timeout = time.Second * 60
		return client
	}
)

type GetClientFunc func(ctx context.Context) *http.Client
type BeforeHook func(context.Context, *http.Request) error

func Download(ctx context.Context, url string, filepath string) error {
	flags := os.O_CREATE | os.O_WRONLY
	f, err := os.OpenFile(filepath, flags, 0666)
	if err != nil {
		fmt.Println("创建文件失败")
		log.Fatal("err")
	}
	defer f.Close()
	_, err = Invoke(ctx, http.MethodGet, url, nil, func(r io.Reader) ([]byte, error) {
		buf := make([]byte, 16*1024)
		_, err = io.CopyBuffer(f, r, buf)
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("io.EOF")
			}
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func BaseSend(ctx context.Context, method string, url string, data interface{}, contentCallback func(string, string) BeforeHook, hooks ...BeforeHook) ([]byte, error) {
	var body io.Reader
	var content string
	if data != nil {
		if m, ok := data.(nUrl.Values); ok {
			content = m.Encode()
			body = strings.NewReader(content)
			hooks = append(hooks, WithURLContentType())
		} else {
			bs, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			content = string(bs)
			body = bytes.NewReader(bs)
			hooks = append(hooks, WithJSONContentType())
		}
		if contentCallback != nil {
			h := contentCallback(url, content)
			if h != nil {
				hooks = append(hooks, h)
			}
		}
	}
	bs, err := Invoke(ctx, method, url, body, nil, hooks...)
	var responseStr string
	if len(bs) > 0 {
		responseStr = string(bs)
	}
	if err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("AgentCall[%s]%s,Request=%s,Response=%s,Error=%s", method, url, content, responseStr, err))
	} else {
		logger.InfoCtx(ctx, fmt.Sprintf("AgentCall[%s]%s,Request=%s,Response=%s,Error=%s", method, url, content, responseStr, err))
	}
	return bs, err
}

func Invoke(ctx context.Context, method string, url string, body io.Reader, respReader func(r io.Reader) ([]byte, error), hooks ...BeforeHook) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if hooks == nil {
		hooks = make([]BeforeHook, 0)
	}
	hooks = append(hooks, WithRequestID())
	for _, hook := range hooks {
		e := hook(ctx, req)
		if e != nil {
			logger.InfoCtx(ctx, e.Error())
		}
	}
	resp, err := DefaultClient(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http code is %d", resp.StatusCode)
	}
	defer func(Body io.ReadCloser) {
		e := Body.Close()
		if e != nil {
			logger.InfoCtx(ctx, e.Error())
		}
	}(resp.Body)
	if respReader != nil {
		return respReader(resp.Body)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return resBody, nil
}

func WithURLContentType() BeforeHook {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return nil
	}
}

func WithJSONContentType() BeforeHook {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Content-Type", "application/json")
		return nil
	}
}

func WithRequestID() BeforeHook {
	return func(ctx context.Context, req *http.Request) error {
		if v := ctx.Value(core.TraceID); v != nil {
			if str, ok := v.(string); ok {
				req.Header.Set("X-Request-ID", str)
			}
		}
		return nil
	}
}
