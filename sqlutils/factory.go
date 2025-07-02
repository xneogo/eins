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
 @Time    : 2025/7/2 -- 13:41
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: sqlutils sqlutils/factory.go
*/

package sqlutils

type Rows interface {
	Close() error
	Columns() ([]string, error)
	Next() bool
	Scan(dest ...interface{}) error
}

// Comparable requires type implements the Build method
type Comparable interface {
	Build() ([]string, []interface{})
}

// ZSqlizer is a wrapper of "github.com/Masterminds/squirrel".Sqlizer
// so we can make some customizes of ToSql function
type ZSqlizer interface {
	ToSql() (string, []interface{}, error)
}
type ToSql func(tName string, columns ...string) (string, []interface{}, error)
