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

package lifecycle

import (
	"context"

	"fx-master/internal/fxlog"
	"fx-master/internal/fxreflect"
	"go.uber.org/multierr"
)

// Hook = <OnStart, OnStop>,任意方法都可以为nil
// 提供Hook的唯一标识
// A Hook is a pair of start and stop callbacks, either of which can be nil,
// plus a string identifying the supplier of the hook.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
	caller  string
}

// 用于协调app中的定义hooks
// Lifecycle coordinates application lifecycle hooks.
type Lifecycle struct {
	logger     *fxlog.Logger   // 操作记录
	hooks      []Hook          // app中开启的hook
	numStarted int             // 已开启的hook???
}

// New constructs a new Lifecycle.
func New(logger *fxlog.Logger) *Lifecycle {  // 创建Liftcycle
	if logger == nil {
		logger = fxlog.New()
	}
	return &Lifecycle{logger: logger}  // 新建Liftcycle并附带logger
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {  // app生命周期中新增新的hook
	hook.caller = fxreflect.Caller()     // 每个调用帧的完整调用链
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
// 启动所有的hook；不过任意一个hook启动过程中产生了error都会导致程序立马结束
func (l *Lifecycle) Start(ctx context.Context) error {
	for _, hook := range l.hooks {
		if hook.OnStart != nil {
			l.logger.Printf("START\t\t%s()", hook.caller)
			if err := hook.OnStart(ctx); err != nil { // 逐一启动hook的Start 并记录到liftcycle的hooks 切片中
				return err
			}
		}
		l.numStarted++  // 记录已完成开启的hook
	}
	return nil
}

// Stop runs any OnStop hooks whose OnStart counterpart succeeded. OnStop
// hooks run in reverse order.
// 停止任意hook(需要当前hook已经启动了start)
func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error
	// Run backward from last successful OnStart.
	for ; l.numStarted > 0; l.numStarted-- {  // 从上一次成功的OnStart处开始 往后处理对应的hook
		hook := l.hooks[l.numStarted-1]  // numStarted记录成功执行的OnStart
		if hook.OnStop == nil {
			continue
		}
		l.logger.Printf("STOP\t\t%s()", hook.caller)
		if err := hook.OnStop(ctx); err != nil {
			// For best-effort cleanup, keep going after errors.
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)  // 输出所有stop失败的hook产生的error
}
