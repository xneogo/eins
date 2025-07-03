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
 @Time    : 2024/10/25 -- 14:21
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: dao.go
 @ currently a copy of xsql.xdao. trying to district behavior
*/

package onesql

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/xneogo/extensions/colorlog"
	"github.com/xneogo/matrix/msql"
	"github.com/xneogo/matrix/msql/sqlutils"
)

type FastDao[DObj any] interface {
	msql.DaoModel[DObj]
}

type FastDaoImpl[DObj any] struct {
	tableName func() string
	omits     func() []string
	_scanner  msql.Scanner
	_builder  msql.Builder
	bind      msql.BindFunc
}

func (dao *FastDaoImpl[DObj]) TableName() string { return dao.tableName() }

func (dao *FastDaoImpl[DObj]) Omits() []string { return dao.omits() }

func (dao *FastDaoImpl[DObj]) Init(cons msql.SqlConstructor, tableName func() string, omits func() []string, b msql.BindFunc) {
	dao._builder = cons.GetBuilder()
	dao._scanner = cons.GetScanner()
	dao.omits = omits
	dao.tableName = tableName
	dao.bind = b
}

func (dao *FastDaoImpl[DObj]) GetScanner() msql.Scanner {
	return dao._scanner
}

func (dao *FastDaoImpl[DObj]) GetBuilder() msql.Builder {
	return dao._builder
}

func (dao *FastDaoImpl[DObj]) SelectOne(ctx context.Context, db msql.XDB, where map[string]interface{}) (res DObj, err error) {
	if nil == db {
		return res, errors.New("manager.XDB object couldn't be nil")
	}
	tar := sqlutils.CopyWhere(where)
	if _, ok := tar["_limit"]; !ok {
		tar["_limit"] = []uint{0, 1}
	}
	cond, vals, err := dao._builder.BuildSelectWithContext(ctx, dao.TableName(), tar, dao.Omits())
	if nil != err {
		return res, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	rows, err := db.QueryContext(ctx, cond, vals...)
	if nil != err || nil == rows {
		return res, err
	}
	defer rows.Close()
	err = dao._scanner.Scan(rows, &res, dao.bind)
	fmt.Println("res", res)
	return res, err
}

func (dao *FastDaoImpl[DObj]) SelectMulti(ctx context.Context, db msql.XDB, where map[string]interface{}) (res []DObj, err error) {
	if nil == db {
		return res, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildSelectWithContext(ctx, dao.TableName(), where, dao.Omits())
	if nil != err {
		return nil, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	rows, err := db.QueryContext(ctx, cond, vals...)
	if nil != err || nil == rows {
		return nil, err
	}
	defer rows.Close()
	err = dao._scanner.Scan(rows, &res, dao.bind)
	return res, err
}

func (dao *FastDaoImpl[DObj]) Insert(ctx context.Context, db msql.XDB, data []map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildInsert(dao.TableName(), data)
	if nil != err {
		return 0, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err || nil == result {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *FastDaoImpl[DObj]) Upsert(ctx context.Context, db msql.XDB, data map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildUpsert(dao.TableName(), data)
	if nil != err {
		return 0, err
	}
	colorlog.Infof(ctx, "build cond: %s vals: %v", cond, vals)
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err || nil == result {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *FastDaoImpl[DObj]) Update(ctx context.Context, db msql.XDB, where, data map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildUpdate(dao.TableName(), where, data)
	if nil != err {
		return 0, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}

func (dao *FastDaoImpl[DObj]) Delete(ctx context.Context, db msql.XDB, where map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildDelete(dao.TableName(), where)
	if nil != err {
		return 0, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}

func (dao *FastDaoImpl[DObj]) CountOf(ctx context.Context, db msql.XDB, where map[string]interface{}) (count int, err error) {
	if nil == db {
		return 0, errors.New("manager.XDB object couldn't be nil")
	}
	cond, vals, err := dao._builder.BuildSelect(dao.TableName(), where, []string{"count(*) as count"})
	if nil != err {
		return 0, err
	}
	colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
	rows, err := db.QueryContext(ctx, cond, vals...)
	if nil != err {
		return 0, err
	}
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return
		}
	}
	return
}

// ComplexQuery
// you can use this default logic or
// you can build your own query logic with or without tableName or columns
// depends on your ToSql func
func ComplexQuery[ans any](tableName string, columns ...string) msql.ComplexQueryMod[ans] {
	return func(
		ctx context.Context,
		db msql.XDB,
		scanner msql.Scanner,
		f msql.ToSql,
		bind msql.BindFunc,
	) (res []ans, err error) {
		if nil == db {
			return nil, errors.New("manager.XDB object couldn't be nil")
		}
		cond, vals, err := f(tableName, columns...)
		if nil != err {
			return nil, err
		}
		colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
		colorlog.Infof(ctx, "build cond: %s vals: %v", cond, vals)
		rows, err := db.QueryContext(ctx, cond, vals...)
		if nil != err || nil == rows {
			return nil, err
		}
		defer rows.Close()
		err = scanner.Scan(rows, &res, bind)
		return res, err
	}
}

func ComplexExec(tableName string) msql.ComplexExecMod {
	return func(
		ctx context.Context,
		db msql.XDB,
		f msql.ToSql,
	) (int64, error) {
		if nil == db {
			return 0, errors.New("manager.XDB object couldn't be nil")
		}
		cond, vals, err := f(tableName)
		if nil != err {
			return 0, err
		}
		colorlog.Debugf(ctx, "build cond: %s vals: %v", cond, vals)
		result, err := db.ExecContext(ctx, cond, vals...)
		if nil != err {
			return 0, err
		}
		return result.RowsAffected()
	}
}
