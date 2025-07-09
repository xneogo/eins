/*
 *  ┏┓      ┏┓
 *┏━┛┻━━━━━━┛┻┓
 *┃　　　━　　  ┃
 *┃   ┳┛ ┗┳   ┃
 *┃           ┃
 *┃     ┻     ┃
 *┗━━━┓     ┏━┛
 *　　 ┃　　　┃神兽保佑
 *　　 ┃　　　┃代码无BUG！
 *　　 ┃　　　┗━━━┓
 *　　 ┃         ┣┓
 *　　 ┃         ┏┛
 *　　 ┗━┓┓┏━━┳┓┏┛
 *　　   ┃┫┫  ┃┫┫
 *      ┗┻┛　 ┗┻┛
 @Time    : 2025/7/9 -- 17:02
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: oneginreq onegin/ginreq/req.go
*/

package ginreq

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/xneogo/eins/onelog"
)

type XRequest struct {
	r       *http.Request
	query   *XKeys
	headers *XKeys
	body    []byte
}

func NewXRequest(r *http.Request) *XRequest {
	return &XRequest{
		r:       r,
		query:   NewXKeys(&Xargs{r: r}),
		headers: NewXKeys(&XHeaders{r: r}),
		body:    nil,
	}
}
func (x *XRequest) Header() http.Header {
	return x.r.Header
}
func (x *XRequest) Body() []byte {
	return x.body
}
func (x *XRequest) Query() *XKeys {
	return x.query
}
func (x *XRequest) Binary() []byte {
	if x.body != nil {
		return x.body
	}
	var err error
	x.body, err = io.ReadAll(x.r.Body)
	if err != nil {
		if err != io.EOF {
			onelog.Ctx(x.r.Context()).Error().Err(err).Msg("read http body error")
		}
		onelog.Ctx(x.r.Context()).Error().Err(err).Msg("read http body error")
	}
	onelog.Ctx(x.r.Context()).Debug().Str("body", string(x.body))
	return x.body
}
func (x *XRequest) BinaryGzip() []byte {
	if x.body != nil {
		return x.body
	}
	fun := "XRequest.BinaryGzip"
	reader, err := gzip.NewReader(x.r.Body)
	if err != nil {
		log.Ctx(x.r.Context()).Error().Err(err).Msg(fun)
	}
	defer reader.Close()

	x.body, err = io.ReadAll(reader)
	if err != nil {
		log.Ctx(x.r.Context()).Error().Err(err).Msg(fun)
	}
	return x.body
}

func JsonWrapper(r *http.Request, js interface{}) error {
	x := NewXRequest(r)
	if x.Query().Int("gzip") == 1 {
		return JsonUnGzip(x, js)
	}
	return Json(x, js)
}

func Json(r *XRequest, js interface{}) error {
	var err error
	dc := json.NewDecoder(bytes.NewBuffer(r.Binary()))
	dc.UseNumber()
	err = dc.Decode(js)
	if err != nil {
		return fmt.Errorf("json unmarshal %s", err.Error())
	} else {
		return nil
	}
}

func JsonUnGzip(r *XRequest, js interface{}) error {
	var err error
	dc := json.NewDecoder(bytes.NewBuffer(r.BinaryGzip()))
	dc.UseNumber()
	err = dc.Decode(js)
	if err != nil {
		return fmt.Errorf("json unmarshal %s", err.Error())
	} else {
		return nil
	}
}
