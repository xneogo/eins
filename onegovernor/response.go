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
 @Time    : 2024/11/4 -- 17:02
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: response.go
*/

package onegovernor

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	htmlContentType     = "text/html; charset=utf-8"
	jsonContentType     = "application/json; charset=utf-8"
	protobufContentType = "application/x-protobuf"
	plainContentType    = "text/plain; charset=utf-8"
	TOMLContentType     = "application/toml; charset=utf-8"
	xmlContentType      = "application/xml; charset=utf-8"
	yamlContentType     = "application/yaml; charset=utf-8"
)

func Status(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
}

func String(w http.ResponseWriter, code int, format string, values ...any) {
	w.WriteHeader(code)
	fmt.Fprintf(w, format, values...)
}

func Json(w http.ResponseWriter, code int, obj any) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", jsonContentType)
	bs, _ := json.Marshal(obj)
	_, _ = w.Write(bs)
}

func Proto(w http.ResponseWriter, code int, obj protoreflect.ProtoMessage) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", protobufContentType)
	bs, _ := proto.Marshal(obj)
	_, _ = w.Write(bs)
}
