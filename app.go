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

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/dig"
	"fx-master/internal/fxlog"
	"fx-master/internal/fxreflect"
	"fx-master/internal/lifecycle"
	"go.uber.org/multierr"
)

// DefaultTimeout is the default timeout for starting or stopping an
// application. It can be configured with the StartTimeout and StopTimeout
// options.
// 控制App在启动和停止过程中的有效时间，保证两个过程能够在有效时间周期内给出结果：执行完成或输出error
//   可以通过StartTimeout和StopTimeout两个选项来进行配置对应的值
const DefaultTimeout = 15 * time.Second

// An Option configures an App using the functional options paradigm
// popularized by Rob Pike. If you're unfamiliar with this style, see
// https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html.

// 在App中所有的function都是以Option提供：errorHookOption、provideOption、invokeOption等
//    采用"函数式选项模式"来进行编程
type Option interface {
	apply(*App)
}

type optionFunc func(*App)

func (f optionFunc) apply(app *App) { f(app) }

// Provide registers any number of constructor functions, teaching the
// application how to instantiate various types. The supplied constructor
// function(s) may depend on other types available in the application, must
// return one or more objects, and may return an error. For example:
//
//  // Constructs type *C, depends on *A and *B.
//  func(*A, *B) *C
//
//  // Constructs type *C, depends on *A and *B, and indicates failure by
//  // returning an error.
//  func(*A, *B) (*C, error)
//
//  // Constructs types *B and *C, depends on *A, and can fail.
//  func(*A) (*B, *C, error)
//
// The order in which constructors are provided doesn't matter, and passing
// multiple Provide options appends to the application's collection of
// constructors. Constructors are called only if one or more of their returned
// types are needed, and their results are cached for reuse (so instances of a
// type are effectively singletons within an application). Taken together,
// these properties make it perfectly reasonable to Provide a large number of
// constructors even if only a fraction of them are used.
//
// See the documentation of the In and Out types for advanced features,
// including optional parameters and named instances.
//
// Provide主要用于完成类型注入名称对应的具体实现类构造函数，任意数量个，用来完成最终实例化变量类型。
// 提供的构造函数有可能会依赖其他类型变量，对于构造函数的返回结果可以是一个对象或多个对象甚至包括error
// 构造类型 *C 其依赖*A 和 *B: 对应的构造函数 func(*A, *B) (*C, error)
// 构造类型 *B 和 *C 依赖类型*A: 对应的构造函数 func(*A) (*B, *C, error)
// 等等
// 不过有一点由于提供的构造函数不止一个，那么其顺序是不能保证的也无需关心的，可以传递多个构造函数作为Option添加到App的构造函数集合中
// 只有在一个或多个构造函数对应的类型被需要时对应的构造函数才会被调用，返回的结果会被缓存下来便于重复利用，也能保证每一种类型以单例的形式存在App的生命周期中，
// 由于这些特性能够保障提供大量的构造函数，即使只有其中一部分被使用。
func Provide(constructors ...interface{}) Option { // 提供构造函数
	return provideOption(constructors)
}

type provideOption []interface{}

func (po provideOption) apply(app *App) {  // 新增新的构造函数
	app.provides = append(app.provides, po...)
}

// Options以字符串形式输出
//   格式: fx.Provide(vender/xxx/xxx.function()，vender/xxx/xxx.function()，...)
func (po provideOption) String() string {
	items := make([]string, len(po))
	for i, c := range po {
		items[i] = fxreflect.FuncName(c)
	}
	return fmt.Sprintf("fx.Provide(%s)", strings.Join(items, ", "))
}

// Invoke registers functions that are executed eagerly on application start.
// Arguments for these invocations are built using the constructors registered
// by Provide. Passing multiple Invoke options appends the new invocations to
// the application's existing list.
//
// Unlike constructors, invocations are always executed, and they're always
// run in order. Invocations may have any number of returned values. If the
// final returned object is an error, it's assumed to be a success indicator.
// All other returned values are discarded.
//
// Typically, invoked functions take a handful of high-level objects (whose
// constructors depend on lower-level objects) and introduce them to each
// other. This kick-starts the application by forcing it to instantiate a
// variety of types.
//
// To see an invocation in use, read through the package-level example. For
// advanced features, including optional parameters and named instances, see
// the documentation of the In and Out types.
// 用于在application start时需要调用register函数
//   对应调用参数是通过Provide函数提供的构造函数来构建的；能通过提供多个Invoke option并将该invocations到已存在的application已存在的列表
// 不同于构造函数，invocation是经常性执行的，并也能经常有序运行。这些invocations会返回不同数量的返回值
// 需要注意若是最终的返回值是一个error，那么就认为当前invocations执行已完成，其他的都将被丢弃。
func Invoke(funcs ...interface{}) Option {
	return invokeOption(funcs)
}

type invokeOption []interface{}

func (io invokeOption) apply(app *App) {
	app.invokes = append(app.invokes, io...)
}

func (io invokeOption) String() string {
	items := make([]string, len(io))
	for i, f := range io {
		items[i] = fxreflect.FuncName(f)
	}
	return fmt.Sprintf("fx.Invoke(%s)", strings.Join(items, ", "))
}

// Error registers any number of errors with the application to short-circuit
// startup. If more than one error is given, the errors are combined into a
// single error.
//
// Similar to invocations, errors are applied in order. All Provide and Invoke
// options registered before or after an Error option will not be applied.
//
// 注册大量在App启动过程中触发short-circuit的错误，将多个error合并到一个error中
// 在Error中对应error都是有序被应用的，这一点和invoke很相似，一个error发生都会导致Provide和Invoke在此之前或之后都将不能使用
func Error(errs ...error) Option {
	return optionFunc(func(app *App) {
		app.err = multierr.Append(app.err, multierr.Combine(errs...))
	})
}

// Options converts a collection of Options into a single Option. This allows
// packages to bundle sophisticated functionality into easy-to-use Fx modules.
// For example, a logging package might export a simple option like this:
//
//  package logging
//
//  var Module = fx.Provide(func() *log.Logger {
//    return log.New(os.Stdout, "", 0)
//  })
//
// A shared all-in-one microservice package could then use Options to bundle
// logging with similar metrics, tracing, and gRPC modules:
//
//  package server
//
//  var Module = fx.Options(
//    logging.Module,
//    metrics.Module,
//    tracing.Module,
//    grpc.Module,
//  )
//
// Since this all-in-one module has a minimal API surface, it's easy to add
// new functionality to it without breaking existing users. Individual
// applications can take advantage of all this functionality with only one
// line of code:
//
//  app := fx.New(server.Module)
//
// Use this pattern sparingly, since it limits the user's ability to customize
// their application.
//
// 将提供的多个Option合并到一个Option中
// 例子：
//  package logging
//
//  var Module = fx.Provide(func() *log.Logger {
//    return log.New(os.Stdout, "", 0)
//  })
// 构建一个一体式的微服务包并将logger module应用到其中
//  package server
//
//  var Module = fx.Options(
//    logging.Module,
//    metrics.Module,
//    tracing.Module,
//    grpc.Module,
//  )
//
// 接下来能够很好的拓展新的功能 而不用影响现有用户的使用
//
//  app := fx.New(server.Module) 一行代码完成App的初始化
//
// 不过上面例子的模式 在使用过程中要尽量限制 这种方式降低使用者的可控范围 限制增强
func Options(opts ...Option) Option { // 目前主要是针对Group属性
	return optionGroup(opts)
}

type optionGroup []Option

func (og optionGroup) apply(app *App) {
	for _, opt := range og {
		opt.apply(app)
	}
}

func (og optionGroup) String() string {
	items := make([]string, len(og))
	for i, opt := range og {
		items[i] = fmt.Sprint(opt)
	}
	return fmt.Sprintf("fx.Options(%s)", strings.Join(items, ", "))
}

// StartTimeout changes the application's start timeout.
// App启动有效时间周期
func StartTimeout(v time.Duration) Option {
	return optionFunc(func(app *App) {
		app.startTimeout = v
	})
}

// App关闭的有效时间周期
// StopTimeout changes the application's stop timeout.
func StopTimeout(v time.Duration) Option {
	return optionFunc(func(app *App) {
		app.stopTimeout = v
	})
}

// 日志接口
// Printer is the interface required by Fx's logging backend. It's implemented
// by most loggers, including the one bundled with the standard library.
type Printer interface {
	Printf(string, ...interface{})
}

// Logger redirects the application's log output to the provided printer.
func Logger(p Printer) Option {
	return optionFunc(func(app *App) {
		app.logger = &fxlog.Logger{Printer: p}
		app.lifecycle = &lifecycleWrapper{lifecycle.New(app.logger)}
	})
}

// NopLogger disables the application's log output. Note that this makes some
// failures difficult to debug, since no errors are printed to console.
// 禁用application的log输出，同时这也让debug变得困难，由于对应的error不能被打印到console(默认fx日志输出到console)
var NopLogger = Logger(nopLogger{})

type nopLogger struct{}

func (l nopLogger) Printf(string, ...interface{}) {
	return
}

// An App is a modular application built around dependency injection. Most
// users will only need to use the New constructor and the all-in-one Run
// convenience method. In more unusual cases, users may need to use the Err,
// Start, Done, and Stop methods by hand instead of relying on Run.
//
// New creates and initializes an App. All applications begin with a
// constructor for the Lifecycle type already registered.
//
// In addition to that built-in functionality, users typically pass a handful
// of Provide options and one or more Invoke options. The Provide options
// teach the application how to instantiate a variety of types, and the Invoke
// options describe how to initialize the application.
//
// When created, the application immediately executes all the functions passed
// via Invoke options. To supply these functions with the parameters they
// need, the application looks for constructors that return the appropriate
// types; if constructors for any required types are missing or any
// invocations return an error, the application will fail to start (and Err
// will return a descriptive error message).
//
// Once all the invocations (and any required constructors) have been called,
// New returns and the application is ready to be started using Run or Start.
// On startup, it executes any OnStart hooks registered with its Lifecycle.
// OnStart hooks are executed one at a time, in order, and must all complete
// within a configurable deadline (by default, 15 seconds). For details on the
// order in which OnStart hooks are executed, see the documentation for the
// Start method.
//
// At this point, the application has successfully started up. If started via
// Run, it will continue operating until it receives a shutdown signal from
// Done (see the Done documentation for details); if started explicitly via
// Start, it will operate until the user calls Stop. On shutdown, OnStop hooks
// execute one at a time, in reverse order, and must all complete within a
// configurable deadline (again, 15 seconds by default).

// App是一个围绕依赖注入的模块化application，大多数用户可以通过新建一个构造函数，并提供Run一体化方法。
// 在很多不寻常的cases，用户可以手动调用Err、Start、Done、Stop等方法替换运行Run。
//
// 新建并初始化App， 所有的Applications都以一个已注册LifeCycle的构造函数开始。
//
// 除了内置的功能，用户可以通过传递一些Provide和一个或多个Invoke选项：Provide选项完成一些不同类型的实例化；Invoke选项来完成初始化application
//
// 当进行创建时，application会立刻执行通过invoke选项提供的函数，为了提供这些函数所需要的参数，application寻找返回对应类型的构造函数：若是所需类型的构造函数丢失或任意invocation返回error，application都将启动失败，Err将返回描述性错误消息
//
// 一旦所有的invocations完成调用(也包括任意需要的构造函数)，新建Application返回接着就会通过Run()或Start()完成启动，当执行启动时，任意的OnStart hook都会注册其各自的LifeCycle
// OnStart hook每次都会执行一次，有序，并且需要在指定的截止时间之前完成(默认15s)。有关OnStart Hook执行顺序的详情，见Start方法文档
//
// 至此application已成功启动，一旦通过Run()启动，application将一直操作直到接收到Done channel发送shutdown信号。若是使用Start()启动，一旦调用Stop()停止操作。shutdown、OnStop每次仅执行一次，不过执行顺序与启动顺序相反，也是必须在指定deadline时间内完成(默认15s)

type App struct {
	err          error
	container    *dig.Container
	lifecycle    *lifecycleWrapper
	provides     []interface{}
	invokes      []interface{}
	logger       *fxlog.Logger
	startTimeout time.Duration
	stopTimeout  time.Duration
	errorHooks   []ErrorHandler

	donesMu sync.RWMutex
	dones   []chan os.Signal
}

// ErrorHook registers error handlers that implement error handling functions.
// They are executed on invoke failures. Passing multiple ErrorHandlers appends
// the new handlers to the application's existing list.
//
// 注册error处理类在执行过程中出现调用失败时能够被执行
// 可以提供多个ErrorHandler并追加到app对应的errorHandlerList([]ErrorHandler)上
func ErrorHook(funcs ...ErrorHandler) Option {
	return errorHookOption(funcs)
}

// ErrorHandler handles Fx application startup errors.
type ErrorHandler interface {
	HandleError(error)
}

type errorHookOption []ErrorHandler // Error处理类

func (eho errorHookOption) apply(app *App) {  // 添加Handler用于处理app出现error时进行的操作
	app.errorHooks = append(app.errorHooks, eho...)
}

type errorHandlerList []ErrorHandler  // app中已添加的所有ErrorHandler

func (ehl errorHandlerList) HandleError(err error) { // 执行具体的Error处理
	for _, eh := range ehl {
		eh.HandleError(err)
	}
}

// New creates and initializes an App, immediately executing any functions
// registered via Invoke options. See the documentation of the App struct for
// details on the application's initialization, startup, and shutdown logic.
//
// 新建并初始化app，并会立刻执行通过invoke选项注册的函数
func New(opts ...Option) *App {
	logger := fxlog.New()   // 日志
	lc := &lifecycleWrapper{lifecycle.New(logger)} // 将application的lifecycle与logger整合 便于记录application的lifecycle

	app := &App{
		container:    dig.New(dig.DeferAcyclicVerification()),  // 容器
		lifecycle:    lc,                                       // app生命周期
		logger:       logger,									// logger
		startTimeout: DefaultTimeout,                           // 启动有效期 (启动app时 完成注册option的执行有效期)
		stopTimeout:  DefaultTimeout,							// 停止有效期 (停止app时 针对完成注册option处理有效期)
	}

	for _, opt := range opts {  // 应用option
		opt.apply(app)
	}

	for _, p := range app.provides { // provide构造函数
		app.provide(p)
	}
	// 三个特殊的provide：Lifecycle/shutdowner/dotGraph
	app.provide(func() Lifecycle { return app.lifecycle })
	app.provide(app.shutdowner)
	app.provide(app.dotGraph)

	if app.err != nil {  // 在App很多内容是以Option提供的 有可能在Option被应用后App出现error 不过这时可以直接返回App 在通过Stop来进行App停止操作
		app.logger.Printf("Error after options were applied: %v", app.err)
		return app
	}

	// 在Option应用过程正常 会对invoke进行执行：通过invoke提供的操作都会被立刻执行 而不会延迟执行
	if err := app.executeInvokes(); err != nil {
		app.err = err  // 执行invoke出现error

		if dig.CanVisualizeError(err) {
			var b bytes.Buffer
			dig.Visualize(app.container, &b, dig.VisualizeError(err))
			err = errorWithGraph{
				graph: b.String(),
				err:   err,
			}
		}
		errorHandlerList(app.errorHooks).HandleError(err)  // 使用errorHandlerList中的ErrorHandler对error进行处理
	}
	return app
}

// DotGraph contains a DOT language visualization of the dependency graph in
// an Fx application. It is provided in the container by default at
// initialization. On failure to build the dependency graph, it is attached
// to the error and if possible, colorized to highlight the root cause of the
// failure.
//
// 提供了一个App可视化依赖发生error的结构图，并对失败根源进行颜色高亮作色，以突显错误根源，在初始化过程中会默认提供
type DotGraph string

type errWithGraph interface {
	Graph() DotGraph
}

type errorWithGraph struct {
	graph string
	err   error
}

func (err errorWithGraph) Graph() DotGraph {
	return DotGraph(err.graph)
}

func (err errorWithGraph) Error() string {
	return err.err.Error()
}

// VisualizeError returns the visualization of the error if available.
// 形象化输出error: 需要error参数属于可用
func VisualizeError(err error) (string, error) {
	if e, ok := err.(errWithGraph); ok && e.Graph() != "" {
		return string(e.Graph()), nil
	}
	return "", errors.New("unable to visualize error")
}

// Run starts the application, blocks on the signals channel, and then
// gracefully shuts the application down. It uses DefaultTimeout to set a
// deadline for application startup and shutdown, unless the user has
// configured different timeouts with the StartTimeout or StopTimeout options.
// It's designed to make typical applications simple to run.
//
// However, all of Run's functionality is implemented in terms of the exported
// Start, Done, and Stop methods. Applications with more specialized needs
// can use those methods directly instead of relying on Run.

// 启动application，并阻塞在signal通道上，来优雅的关闭app。
//
// 通过使用DefaultTimeout来设置app的启动和关闭deadline，也可以通过StartTimeout和StopTimeout选项来进行设置，DefaultTimeout能够保证app简单执行
//
// Run()是整合了Start()、Done()、Stop()的功能，有更特殊需求的app可以直接使用这些方法，而不是依赖于Run
func (app *App) Run() {
	app.run(app.Done())
}

// Err returns any error encountered during New's initialization. See the
// documentation of the New method for details, but typical errors include
// missing constructors, circular dependencies, constructor errors, and
// invocation errors.
//
// Most users won't need to use this method, since both Run and Start
// short-circuit if initialization failed.

// 在执行New()初始化期间返回任何发生的error
// 该方法并不是必须使用的 因为在Run和Start初始化失败时都会short-circuit
func (app *App) Err() error {
	return app.err
}

// Start kicks off all long-running goroutines, like network servers or
// message queue consumers. It does this by interacting with the application's
// Lifecycle.
//
// By taking a dependency on the Lifecycle type, some of the user-supplied
// functions called during initialization may have registered start and stop
// hooks. Because initialization calls constructors serially and in dependency
// order, hooks are naturally registered in dependency order too.
//
// Start executes all OnStart hooks registered with the application's
// Lifecycle, one at a time and in order. This ensures that each constructor's
// start hooks aren't executed until all its dependencies' start hooks
// complete. If any of the start hooks return an error, Start short-circuits,
// calls Stop, and returns the inciting error.
//
// Note that Start short-circuits immediately if the New constructor
// encountered any errors in application initialization.

//
// 启动长时间运行的goroutine，类似network server或消息队列消费，主要是通过与App的Lifecycle进行交互的
//
func (app *App) Start(ctx context.Context) error {
	return withTimeout(ctx, app.start)
}

// Stop gracefully stops the application. It executes any registered OnStop
// hooks in reverse order, so that each constructor's stop hooks are called
// before its dependencies' stop hooks.
//
// If the application didn't start cleanly, only hooks whose OnStart phase was
// called are executed. However, all those hooks are executed, even if some
// fail.
func (app *App) Stop(ctx context.Context) error {
	return withTimeout(ctx, app.lifecycle.Stop)
}

// Done returns a channel of signals to block on after starting the
// application. Applications listen for the SIGINT and SIGTERM signals; during
// development, users can send the application SIGTERM by pressing Ctrl-C in
// the same terminal as the running process.
//
// Alternatively, a signal can be broadcast to all done channels manually by
// using the Shutdown functionality (see the Shutdowner documentation for details).

// 在启动application后返回一个阻塞的signals的channel，app会监听SIGINT和SIGTERM信号，主要是针对app是通过Run()启动
// 一旦启动了 就会一直处理直至通过Done获取signal才会停止
// 在开发期间可以通过对控制台执行ctrl+c 发送SIGTERM信息，也可以将一个signal通过Shutdown的功能手动广播给所有done channels
func (app *App) Done() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	app.donesMu.Lock()
	app.dones = append(app.dones, c)
	app.donesMu.Unlock()
	return c
}

// StartTimeout returns the configured startup timeout. Apps default to using
// DefaultTimeout, but users can configure this behavior using the
// StartTimeout option.
//
// 设置App启动过程的有效时效；默认使用DefaultTimeout
func (app *App) StartTimeout() time.Duration {
	return app.startTimeout
}

// StopTimeout returns the configured shutdown timeout. Apps default to using
// DefaultTimeout, but users can configure this behavior using the StopTimeout
// option.
//
// 设置App关闭过程的有效时效；默认使用DefaultTimeout
func (app *App) StopTimeout() time.Duration {
	return app.stopTimeout
}

// 生成App启动过程的依赖关系图
func (app *App) dotGraph() (DotGraph, error) {
	var b bytes.Buffer
	err := dig.Visualize(app.container, &b)
	return DotGraph(b.String()), err
}

// 添加初始化实例的构造函数 完成注入对象名与其关联具体类
// 注意：provide接收的是function而非Option
func (app *App) provide(constructor interface{}) {
	if app.err != nil {
		return
	}
	app.logger.PrintProvide(constructor)

	if _, ok := constructor.(Option); ok { //
		app.err = fmt.Errorf("fx.Option should be passed to fx.New directly, not to fx.Provide: fx.Provide received %v", constructor)
		return
	}

	if a, ok := constructor.(Annotated); ok { // Annotated类型
		var opts []dig.ProvideOption
		switch {
		case len(a.Group) > 0 && len(a.Name) > 0:  // Group与Name只能设置其中一个
			app.err = fmt.Errorf("fx.Annotate may not specify both name and group for %v", constructor)
			return
		case len(a.Name) > 0:  // 设置Name
			opts = append(opts, dig.Name(a.Name))
		case len(a.Group) > 0:  // 设置Group
			opts = append(opts, dig.Group(a.Group))

		}

		if err := app.container.Provide(a.Target, opts...); err != nil { // 向container提供constructor
			app.err = err
		}
		return
	}

	// 非Annotated 且返回值也不是Annotated
	if reflect.TypeOf(constructor).Kind() == reflect.Func {  // 检查function返回值是否=Annotated
		ft := reflect.ValueOf(constructor).Type()

		for i := 0; i < ft.NumOut(); i++ {
			t := ft.Out(i)

			if t == reflect.TypeOf(Annotated{}) { // 返回值不能使用Annotated
				app.err = fmt.Errorf("fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor: fx.Provide received %v", constructor)
				return
			}
		}
	}

	if err := app.container.Provide(constructor); err != nil {  // 向container提供constructor
		app.err = err
	}
}

// Execute invokes in order supplied to New, returning the first error
// encountered.
//
// 通过invoke提供的function有序执行，且不同于provide提供的function延迟执行，invoke会被立即执行的
//  在执行invoke过程抛出error 则直接返回第一个出现的error返回
func (app *App) executeInvokes() error {
	// TODO: consider taking a context to limit the time spent running invocations.
	var err error

	for _, fn := range app.invokes {  // 遍历invoke
		fname := fxreflect.FuncName(fn)  // 通过反射的方式获取完整function的完整路径：类似vender/xxx/xxx/xxx.function()
		app.logger.Printf("INVOKE\t\t%s", fname)

		if _, ok := fn.(Option); ok { // invoke提供的是function而非Option
			err = fmt.Errorf("fx.Option should be passed to fx.New directly, not to fx.Invoke: fx.Invoke received %v", fn)
		} else {
			err = app.container.Invoke(fn) // container invoke the function
		}

		if err != nil {
			app.logger.Printf("Error during %q invoke: %v", fname, err)
			break
		}
	}

	return err
}

// 启动app执行注入操作  接收signal信号判断是否完成: 等价于OnStart、OnStop的结合体
func (app *App) run(done <-chan os.Signal) {
	startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout()) //
	defer cancel()

	if err := app.Start(startCtx); err != nil {  // start the application
		app.logger.Fatalf("ERROR\t\tFailed to start: %v", err)
	}

	app.logger.PrintSignal(<-done)   // send the done signal ， the app start is completed.

	stopCtx, cancel := context.WithTimeout(context.Background(), app.StopTimeout()) // stop the application
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {  // when the start is completed， the app need to execute stop operation
		app.logger.Fatalf("ERROR\t\tFailed to stop cleanly: %v", err)
	}
}

// app启动：
func (app *App) start(ctx context.Context) error {
	if app.err != nil {
		// Some provides failed, short-circuit immediately.
		return app.err
	}

	// Attempt to start cleanly.
	if err := app.lifecycle.Start(ctx); err != nil {  // 通过app的lifecycle启动 若是启动失败则进行回滚并记录错误现场
		// Start failed, roll back.
		app.logger.Printf("ERROR\t\tStart failed, rolling back: %v", err)
		if stopErr := app.lifecycle.Stop(ctx); stopErr != nil {  // 通过app的lifecycle进行关闭
			app.logger.Printf("ERROR\t\tCouldn't rollback cleanly: %v", stopErr)
			return multierr.Append(err, stopErr)
		}
		return err
	}

	app.logger.Printf("RUNNING")
	return nil
}

func withTimeout(ctx context.Context, f func(context.Context) error) error {
	c := make(chan error, 1)
	go func() { c <- f(ctx) }()  // 开启goroutine执行function，并将结果放置到context.Context

	select {
	case <-ctx.Done():  // 等待执行结果: 正常完成 或诱发错误
		return ctx.Err()
	case err := <-c:
		return err
	}
}
