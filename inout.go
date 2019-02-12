// Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fx

import "go.uber.org/dig"

// fx.In能够被嵌套在构造函数参数结构以获取依赖注入的高级特性
// In can be embedded in a constructor's parameter struct to take advantage of
// advanced dependency injection features.
//
// Modules提供一个具有可正向兼容的API的参数结构，也由于添加新的field在struct中是为了向后兼容，modules能够添加一些可选依赖在一些minor的版本
// Modules should take a single parameter struct that embeds an In in order to
// provide a forward-compatible API: since adding fields to a struct is
// backward-compatible, modules can then add optional dependencies in minor
// releases.
//
// 1、参数结构
//
// 由于Fx constructors声明的依赖是以函数参数的方式，这样可能会带来一旦constructor具有很多依赖时变得难以阅读
// 比如 func NewHandler(users *UserGateway, comments *CommentGateway, posts *PostGateway, votes *VoteGateway, authz *AuthZGateway) *Handler {
//     		...
//	    }
//
//  为了提供类似构造函数的可阅读性，通过创建一个struct将所有依赖作为其field并调整function接受一个struct而非前面的那么多依赖项，这也称之为参数结构
//  Fx框架提供了对参数结构的支持：将fx.In内嵌到任意struct中这样该struct就被称为参数结构，而在这个struct中的field也是通过依赖注入提供具体的值。
//  通过使用参数结构能让constructor变得可读性更强、更清晰
//  使用参数结构的方式
//   type HandlerParams struct {
//     fx.In
//
//     Users    *UserGateway
//     Comments *CommentGateway
//     Posts    *PostGateway
//     Votes    *VoteGateway
//     AuthZ    *AuthZGateway
//   }
//  对应的constructor变成如下的声明：
//   func NewHandler(p HandlerParams) *Handler {
//     // ...
//   }
//
// 2、可选依赖
// 有时constructor中的一些依赖类型属于soft依赖：若是这些依赖类型miss了，那么也不影响参数结构继续被使用
// 针对于该情况Fx框架提供可选依赖通过对参数结构中的field使用`optional:"true"`标签即可达到上述的需求，
//   但凡会被提供`optional:"true"`标签的field是否丢失都不能应该参数结构的使用
// 例如：
//   type UserGatewayParams struct {
//     fx.In
//
//     Conn  *sql.DB
//     Cache *redis.Client `optional:"true"`  // 该字段是否丢失都不影响最终的依赖注入结果的可用
//   }
//
// 一个可选Field在container不可用时，在constructor被使用时会通过其零值来填充，constructor需要能够保证一些可选依赖不可用时提供优雅的解决方案
// 验证函数
//   func NewUserGateway(p UserGatewayParams, log *log.Logger) (*UserGateway, error) {
//     if p.Cache != nil {
//       log.Print("Caching disabled")
//     }
//     // ...
//   }
//
// 同时能够通过使用`optional:"true"`来增加一些新的依赖选项而不会影响到constructor当前已使用方
//
// 3、named values
// 有时一些实例可能需要Application container保存相同类型的多个值，那么就可以使用 `name:".."`标签来完成
//
//   type GatewayParams struct {
//     fx.In
//
//     WriteToConn  *sql.DB `name:"rw"`
//     ReadFromConn *sql.DB `name:"ro"`
//   }
// 同时也能跟`optional:"true"`标签一起使用
//   type GatewayParams struct {
//     fx.In
//
//     WriteToConn  *sql.DB `name:"rw"`
//     ReadFromConn *sql.DB `name:"ro" optional:"true"`
//   }
//
// 4、Value Groups
// 为了支持更多相同类型的值的生成和使用，Fx框架提供`group:".."`标签
// 例如
//   type ServerParams struct {
//     fx.In
//
//     Handlers []Handler `group:"server"`
//   }
//
//   func NewServer(p ServerParams) *Server {
//     server := newServer()
//     for _, h := range p.Handlers {
//       server.Register(h)
//     }
//     return server
//   }
//
// 注意：在group内是无序的
//
// Parameter Structs
//
// Fx constructors declare their dependencies as function parameters. This can
// quickly become unreadable if the constructor has a lot of dependencies.
//
//   func NewHandler(users *UserGateway, comments *CommentGateway, posts *PostGateway, votes *VoteGateway, authz *AuthZGateway) *Handler {
//     // ...
//   }
//
// To improve the readability of constructors like this, create a struct that
// lists all the dependencies as fields and change the function to accept that
// struct instead. The new struct is called a parameter struct.
//
// Fx has first class support for parameter structs: any struct embedding
// fx.In gets treated as a parameter struct, so the individual fields in the
// struct are supplied via dependency injection. Using a parameter struct, we
// can make the constructor above much more readable:
//
//   type HandlerParams struct {
//     fx.In
//
//     Users    *UserGateway
//     Comments *CommentGateway
//     Posts    *PostGateway
//     Votes    *VoteGateway
//     AuthZ    *AuthZGateway
//   }
//
//   func NewHandler(p HandlerParams) *Handler {
//     // ...
//   }
//
// Though it's rarely a good idea, constructors can receive any combination of
// parameter structs and parameters.
//
//   func NewHandler(p HandlerParams, l *log.Logger) *Handler {
//     // ...
//   }
//
// Optional Dependencies
//
// Constructors often have soft dependencies on some types: if those types are
// missing, they can operate in a degraded state. Fx supports optional
// dependencies via the `optional:"true"` tag to fields on parameter structs.
//
//   type UserGatewayParams struct {
//     fx.In
//
//     Conn  *sql.DB
//     Cache *redis.Client `optional:"true"`
//   }
//
// If an optional field isn't available in the container, the constructor
// receives the field's zero value.
//
//   func NewUserGateway(p UserGatewayParams, log *log.Logger) (*UserGateway, error) {
//     if p.Cache != nil {
//       log.Print("Caching disabled")
//     }
//     // ...
//   }
//
// Constructors that declare optional dependencies MUST gracefully handle
// situations in which those dependencies are absent.
//
// The optional tag also allows adding new dependencies without breaking
// existing consumers of the constructor.
//
// Named Values
//
// Some use cases require the application container to hold multiple values of
// the same type. For details on producing named values, see the documentation
// for the Out type.
//
// Fx allows functions to consume named values via the `name:".."` tag on
// parameter structs. Note that both the name AND type of the fields on the
// parameter struct must match the corresponding result struct.
//
//   type GatewayParams struct {
//     fx.In
//
//     WriteToConn  *sql.DB `name:"rw"`
//     ReadFromConn *sql.DB `name:"ro"`
//   }
//
// The name tag may be combined with the optional tag to declare the
// dependency optional.
//
//   type GatewayParams struct {
//     fx.In
//
//     WriteToConn  *sql.DB `name:"rw"`
//     ReadFromConn *sql.DB `name:"ro" optional:"true"`
//   }
//
//   func NewCommentGateway(p GatewayParams, log *log.Logger) (*CommentGateway, error) {
//     if p.ReadFromConn == nil {
//       log.Print("Warning: Using RW connection for reads")
//       p.ReadFromConn = p.WriteToConn
//     }
//     // ...
//   }
//
// Value Groups
//
// To make it easier to produce and consume many values of the same type, Fx
// supports named, unordered collections called value groups. For details on
// producing value groups, see the documentation for the Out type.
//
// Functions can depend on a value group by requesting a slice tagged with
// `group:".."`. This will execute all constructors that provide a value to
// that group in an unspecified order, then collect all the results into a
// single slice. Keep in mind that this makes the types of the parameter and
// result struct fields different: if a group of constructors each returns
// type T, parameter structs consuming the group must use a field of type []T.
//
//   type ServerParams struct {
//     fx.In
//
//     Handlers []Handler `group:"server"`
//   }
//
//   func NewServer(p ServerParams) *Server {
//     server := newServer()
//     for _, h := range p.Handlers {
//       server.Register(h)
//     }
//     return server
//   }
//
// Note that values in a value group are unordered. Fx makes no guarantees
// about the order in which these values will be produced.
type In struct{ dig.In }

// Fx.Out是Fx.In相反面。
// 1、结果结构Result Structs
// 结果结构是相对参数结构来说的：将多个输出结果作为一个struct的fields输出
// Fx将所有的内嵌fx.Out的struct作为结果结构，这样其他constructors能够直接依赖结果结构的fields
// 不适用结果结构的函数声明：
//   func SetupGateways(conn *sql.DB) (*UserGateway, *CommentGateway, *PostGateway, error) {
//     // ...
//   }
// 使用结果结构的函数声明：
//  type Gateways struct {  // 将输出结果合并到一个struct
//    fx.Out
//
//    Users    *UserGateway
//    Comments *CommentGateway
//    Posts    *PostGateway
//  }
//
//  func SetupGateways(conn *sql.DB) (Gateways, error) {
//    // ...
//  }
//
// 2、Named Values
// 有时可能需要具有相同类型的多个值，Fx提供了`name:".."`标签来将相应的值添加到对应的name下，
//   type ConnectionResult struct {
//     fx.Out
//
//     ReadWrite *sql.DB `name:"rw"`
//     ReadOnly  *sql.DB `name:"ro"`
//   }
//
//   func ConnectToDatabase(...) (ConnectionResult, error) {
//     // ...
//     return ConnectionResult{ReadWrite: rw, ReadOnly:  ro}, nil
//   }
//
// 3、Value Groups
// 为了支持更多相同类型的值的生成和使用，Fx框架提供`group:".."`标签
// 例如
//   type HandlerResult struct {
//     fx.Out
//
//     Handler Handler `group:"server"`
//   }
//
//   func NewHelloHandler() HandlerResult {
//     // ...
//   }
//
//   func NewEchoHandler() HandlerResult {
//     // ...
//   }
//
// 注意：在group内是无序的
//
// Out is the inverse of In: it can be embedded in result structs to take
// advantage of advanced features.
//
// Modules should return a single result struct that embeds an Out in order to
// provide a forward-compatible API: since adding fields to a struct is
// backward-compatible, minor releases can provide additional types.
//
// Result Structs
//
// Result structs are the inverse of parameter structs (discussed in the In
// documentation). These structs represent multiple outputs from a
// single function as fields. Fx treats all structs embedding fx.Out as result
// structs, so other constructors can rely on the result struct's fields
// directly.
//
// Without result structs, we sometimes have function definitions like this:
//
//   func SetupGateways(conn *sql.DB) (*UserGateway, *CommentGateway, *PostGateway, error) {
//     // ...
//   }
//
// With result structs, we can make this both more readable and easier to
// modify in the future:
//
//  type Gateways struct {
//    fx.Out
//
//    Users    *UserGateway
//    Comments *CommentGateway
//    Posts    *PostGateway
//  }
//
//  func SetupGateways(conn *sql.DB) (Gateways, error) {
//    // ...
//  }
//
// Named Values
//
// Some use cases require the application container to hold multiple values of
// the same type. For details on consuming named values, see the documentation
// for the In type.
//
// A constructor that produces a result struct can tag any field with
// `name:".."` to have the corresponding value added to the graph under the
// specified name. An application may contain at most one unnamed value of a
// given type, but may contain any number of named values of the same type.
//
//   type ConnectionResult struct {
//     fx.Out
//
//     ReadWrite *sql.DB `name:"rw"`
//     ReadOnly  *sql.DB `name:"ro"`
//   }
//
//   func ConnectToDatabase(...) (ConnectionResult, error) {
//     // ...
//     return ConnectionResult{ReadWrite: rw, ReadOnly:  ro}, nil
//   }
//
// Value Groups
//
// To make it easier to produce and consume many values of the same type, Fx
// supports named, unordered collections called value groups. For details on
// consuming value groups, see the documentation for the In type.
//
// Constructors can send values into value groups by returning a result struct
// tagged with `group:".."`.
//
//   type HandlerResult struct {
//     fx.Out
//
//     Handler Handler `group:"server"`
//   }
//
//   func NewHelloHandler() HandlerResult {
//     // ...
//   }
//
//   func NewEchoHandler() HandlerResult {
//     // ...
//   }
//
// Any number of constructors may provide values to this named collection, but
// the ordering of the final collection is unspecified. Keep in mind that
// value groups require parameter and result structs to use fields with
// different types: if a group of constructors each returns type T, parameter
// structs consuming the group must use a field of type []T.
type Out struct{ dig.Out }
