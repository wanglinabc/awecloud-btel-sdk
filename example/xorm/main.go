// Copyright The AliyunSLS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-gonic/gin"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"xorm.io/xorm"
)

func main() {
	if tracer := btrace.New(); tracer != nil {
		defer tracer.Shutdown()
	}
	db, err := NewEngineForHook()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	r := gin.Default()
	r.Use(otelgin.Middleware(os.Getenv("BTEL_SERVICE_NAME")))
	r.GET("/user/:name", func(c *gin.Context) {
		data, err := query(c.Request.Context())
		if err != nil {
			c.String(500, err.Error())
		} else {
			c.JSON(200, data)
		}
	})
	r.Run(":8080")
}

// xorm 1.0.2已经支持Hook钩子函数注入操作上下文
func NewEngineForHook() (engine *xorm.Engine, err error) {
	// XORM创建引擎
	engine, err = xorm.NewEngine("mysql", "test:password@(localhost:1306)/test?charset=utf8mb4")
	if err != nil {
		return
	}

	engine.ShowSQL(true)
	// 使用我们的钩子函数
	btrace.WrapEngine(engine, otel.Tracer("xorm sql execute"))
	return
}

type Reviews struct {
	Id       int    `json:"id"`
	Text     string `json:"text"`
	Reviewer string `json:"reviewer"`
}

func query(ctx context.Context) (res interface{}, err error) {
	db, err := NewEngineForHook()
	if err != nil {
		return
	}
	// 生成新的Span - 注意将span结束掉，不然无法发送对应的结果
	// span := trace.SpanFromContext(ctx)
	// tracer := otel.Tracer("test-tracer")
	// _, iSpan := tracer.Start(ctx, "xxxxx")
	// defer iSpan.End()
	// 将子上下文传入Session
	/*session := db.Context(ctx)
	u := []Reviews{}
	err = session.Table("reviews").Where("id = 1").Find(&u)
	return u, err*/

	comment := &Reviews{
		Id:       3,
		Text:     "虚竹实是金大侠小说中之第一人。",
		Reviewer: "评论3",
	}

	session := db.Context(ctx)
	_, err = session.Insert(comment)
	if err != nil {
		fmt.Println(err)
	}
	return comment, err
}
