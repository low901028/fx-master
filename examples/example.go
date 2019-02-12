// Copyright 2015 The etcd Authors
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
	"fx-master"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func test1(){
	var reader io.Reader

	app := fx.New(
		// io.reader的应用
		fx.Provide(func() io.Reader { return strings.NewReader("hello world") }),  // 提供构造函数
		fx.Populate(&reader), // 通过依赖注入完成变量与具体类的映射
	)
	app.Start(context.Background())
	defer app.Stop(context.Background())

	// 使用
	bs, err := ioutil.ReadAll(reader)  // reader变量已与fx.Provide注入的实现类关联了
	if err != nil{
		log.Panic("read occur error, ", err)
	}
	fmt.Printf("the result is '%s' \n", string(bs))
}

func test2(){
	type t3 struct {
		Name string
	}

	type t4 struct {
		Age int
	}

	var (
		v1 *t3
		v2 *t4
	)

	app := fx.New(
		fx.Provide(func() *t3 { return &t3{"hello everybody!!!"} }),
		fx.Provide(func() *t4 { return &t4{2019} }),

		fx.Populate(&v1),
		fx.Populate(&v2),
	)

	app.Start(context.Background())
	defer app.Stop(context.Background())

	fmt.Printf("the reulst is %v , %v\n", v1.Name, v2.Age)
}

func test3(){
	type t3 struct {
		Name string
	}
	//name标签的使用
	type result struct {
		fx.Out

		V1 *t3 `name:"n1"`
		V2 *t3 `name:"n2"`
	}

	targets := struct {
		fx.In

		V1 *t3 `name:"n1"`
		V2 *t3 `name:"n2"`
	}{}

	app := fx.New(
		fx.Provide(func() result {
			return result{
				V1: &t3{"hello-HELLO"},
				V2: &t3{"world-WORLD"},
			}
		}),

		fx.Populate(&targets),
	)

	app.Start(context.Background())
	defer app.Stop(context.Background())

	fmt.Printf("the result is %v, %v \n", targets.V1.Name, targets.V2.Name)
}

func test4(){
	type t3 struct {
		Name string
	}

	// 使用group标签
	type result struct {
		fx.Out

		V1 *t3 `group:"g"`
		V2 *t3 `group:"g"`
	}

	targets := struct {
		fx.In

		Group []*t3 `group:"g"`
	}{}

	app := fx.New(
		fx.Provide(func() result {
			return result{
				V1: &t3{"hello-000"},
				V2: &t3{"world-www"},
			}
		}),

		fx.Populate(&targets),
	)

	app.Start(context.Background())
	defer app.Stop(context.Background())

	for _,t := range targets.Group{
		fmt.Printf("the result is %v\n", t.Name)
	}
}

func test5(){
	type t3 struct {
		Name string
	}

	targets := struct {
		fx.In

		V1 *t3 `name:"n1"`
	}{}

	app := fx.New(
		fx.Provide(fx.Annotated{
			Name:"n1",
			Target: func() *t3{
				return &t3{"hello world"}
		    },
		}),
		fx.Populate(&targets),
	)
	app.Start(context.Background())
	defer app.Stop(context.Background())

	//app.Run()

	fmt.Printf("the result is = '%v'\n", targets.V1.Name)
	//<- app.Done()
}

// ====================================分割线==================================
// Logger构造函数
func NewLogger() *log.Logger {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	logger.Print("Executing NewLogger.")
	return logger
}

// http.Handler构造函数
func NewHandler(logger *log.Logger) (http.Handler, error) {
	logger.Print("Executing NewHandler.")
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		logger.Print("Got a request.")
	}), nil
}

// http.ServeMux构造函数
func NewMux(lc fx.Lifecycle, logger *log.Logger) *http.ServeMux {
	logger.Print("Executing NewMux.")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	lc.Append(fx.Hook{ // 使用Hook 重新实现OnStart和OnStop
		OnStart: func(context.Context) error {
			logger.Print("Starting HTTP server.")
			go server.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Print("Stopping HTTP server.")
			return server.Shutdown(ctx)
		},
	})

	return mux
}

// 注册http.Handler
func Register(mux *http.ServeMux, h http.Handler) {
	fmt.Println("Register start")
	mux.Handle("/", h)
	fmt.Println("Register end")
}

func test6(){
	app := fx.New(
		fx.Provide(
			NewLogger,
			NewHandler,
			NewMux,
		),
		fx.Invoke(Register),
	)
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		log.Fatal(err)
	}

	//app.Run()

	http.Get("http://localhost:8080/")

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		log.Fatal(err)
	}
}



type Connection struct {
	Name string
}
func NewReadOnlyConnection() *Connection{
	log.Println("New ReadOnly Connection.")
	return &Connection{"hello world"}
}
func test7(){
	type result struct {
		fx.In

		conn *Connection `name:"ro"`
	}

	var res result

	app := fx.New(
		fx.Provide(
			fx.Annotated{
				Name:"ro",
				Target: NewReadOnlyConnection,
			}),
		fx.Populate(&res),
	)
	//
	app.Start(context.Background())
	defer app.Stop(context.Background())

	fmt.Println(res.conn.Name)
}

func main() {
	test7()
}
