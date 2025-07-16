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
 @Time    : 2024/11/14 -- 17:16
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: u.go
*/

package utils

import (
	"strings"
)

var whiteListUri = []string{
	// "/route/installid/",
	// "/ugc/live/activity/",
	// "/ugc/curriculum/contractinfo/",
	// "/proxy/courseware/sheet/resource/",
	// "/proxy/courseware/sheet/logic/",
	// "/order/wxpaycallback2/",
	// "/order/paypal/callback/approval/",
	// "/order/paypal/callback/cancel/",
	// "/wechatsystem/notify/",
	// "/account/user/",
	// "/account/user2/",
	// "/account/phone/",
	// "/userlevel/getlevel/",
	// "/im/msginfo/",
	// "/im/sendmulti/",
	// "/teacher/privilege/",
	// "/teacher/status/",
	// "/teacher/labels/",
	// "/app/config/",
	// "/ugc/curriculum/base/courseware/uploadremark/",
	// "/honour/order/import/data/",
	// "/honour/order/export/data/",
	// "/honour/order/import/exchange/",
	// "/wechatsystem/wechatauth/upload/openid/",
	// "/filter/bind/uid/upload/",
}

func ParseUriApi(uri string) string {
	for _, u := range whiteListUri {
		if strings.HasPrefix(uri, u) {
			return u
		}
	}
	return uri
}
