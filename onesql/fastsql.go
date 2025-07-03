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
 @Time    : 2024/10/11 -- 18:31
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: fastsql.go
*/

package onesql

import (
	"context"
	"database/sql"

	"github.com/xneogo/matrix/msql"
)

var Constructor *constructor

type constructor struct {
	_scanner *fastScanner
	_builder *fastBuilder
}

func init() {
	Constructor = &constructor{
		_scanner: &fastScanner{},
		_builder: &fastBuilder{},
	}
}

func (c *constructor) GetBuilder() msql.Builder {
	return c._builder
}

func (c *constructor) GetScanner() msql.Scanner {
	return c._scanner
}

func (c *constructor) ComplexSelect(dbx *sql.DB, builder msql.MSqlizer, target any, bind msql.BindFunc) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		query, args, err := builder.ToSql()
		if err != nil {
			return err
		}
		rows, err := dbx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		if _, err := bind(rows); err != nil {
			return err
		}
		return err
	}
}

func (c *constructor) ComplexExec(dbx *sql.DB, builder msql.MSqlizer) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		query, args, err := builder.ToSql()
		if err != nil {
			return err
		}
		_, err = dbx.ExecContext(ctx, query, args...)
		return err
	}
}

// FastRepo only for convenient, a wrapper of msql interfaces
type FastRepo[EntityObj any] interface {
	msql.RepoModel[EntityObj]
}

type FastQueryRequest[EntityObj any] interface {
	msql.QueryRequest[EntityObj]
}

type FastInsertRequest[EntityObj any] interface {
	msql.InsertRequest[EntityObj]
	msql.UpsertRequest[EntityObj]
}

type FastUpdateRequest interface {
	msql.UpdateRequest
}

type FastComplexRequest[EntityObj any] interface {
	msql.ComplexRequest[EntityObj]
}
