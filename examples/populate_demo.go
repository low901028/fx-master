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
)

func main() {
	type t1 struct {
		buf [1024]byte
	}

	type t2 struct{}
	// var v1 *t1
	// var reader io.Reader

	type t3 struct {
		Name string
	}

	type t4 struct {
		Age int
	}

	//var (
	//	v1 *t3
	//	v2 *t4
	//	v3 *t4
	//)

	// struct参数模式
	//targets := struct {
	//	fx.In
	//
	//	V1 *t3
	//	V2 *t4
	//}{}

	// name标签的使用
	//type result struct {
	//	fx.Out
	//
	//	V1 *t3 `name:"n1"`
	//	V2 *t3 `name:"n2"`
	//}
	//
	//targets := struct {
	//	fx.In
	//
	//	V1 *t3 `name:"n1"`
	//	V2 *t3 `name:"n2"`
	//}{}

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
		//fx.Provide(func() *t1 { panic("should not be called ") }),
		//fx.Populate(),

		//fx.Provide(func() *t1 { return &t1{} }),
		//fx.Populate(&v1),

		// io.reader的应用
		//fx.Provide(func() io.Reader { return strings.NewReader("hello world") }),  // 提供构造函数
		//fx.Populate(&reader), // 通过依赖注入完成变量与具体类的映射

		// 模拟两个struct
		//fx.Provide(func() *t3 { return &t3{"hello everybody!!!"} }),
		//fx.Provide(func() *t4 { return &t4{2019} }),
		//
		//fx.Populate(&v1),
		//fx.Populate(&v2),

		// 注入到container构造函数是不能相同的 否则会导致Provide抛出panic
		//fx.Provide(func() *t3 { return &t3{"hello everybody!!!"} },func() *t4 { return &t4{2019} }, /*func() *t4 { return &t4{9012} }*/),
		//fx.Populate(&v2,&v1,&v3),

		// 使用struct参数方式
		//fx.Provide(func() *t3 { return &t3{"hello everybody!!!"} },func() *t4 { return &t4{2019} },),
		//fx.Populate(&targets),

		// 使用struct参数(输入 输出) 可通过name来保证相同类型多个值存放到container中
		//fx.Provide(func() result {
		//	return result{
		//		V1: &t3{"hello-HELLO"},
		//		V2: &t3{"world-WORLD"},
		//	}
		//}),
		//
		//fx.Populate(&targets),

		// 使用group（注意标签name和group两者只能选其一）
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

	// fmt.Printf("the result is %v, %v \n", targets.V1.Name, targets.V2.Name)

	//fmt.Printf("the reulst is %v , %v\n", targets.V1.Name, targets.V2.Age)
	//fmt.Printf("the reulst is %v , %v, %v\n", v1.Name, v2.Age, v3.Age)
	// io.reader的应用
	//bs, err := ioutil.ReadAll(reader)
	//if err != nil{
	//	log.Panic("read occur error, ", err)
	//}
	//fmt.Printf("the result is '%s' \n", string(bs))
}
