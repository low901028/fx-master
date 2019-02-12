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
	"fmt"
	"os"
	"syscall"
)

// 提供了手动触发application的shutdown，发送一个signal信号给所有处于open的Done-channel
// 不过Shutdowner使用需要application是用Run方法来启动的(兼顾了Start、Done、Stop等操作)
// Shutdowner provides a method that can manually trigger the shutdown of the
// application by sending a signal to all open Done channels. Shutdowner works
// on applications using Run as well as Start, Done, and Stop. The Shutdowner is
// provided to all Fx applications.
type Shutdowner interface {
	Shutdown(...ShutdownOption) error
}

// 提供shutdowm相关处理的配置属性
// 注意：当前没有option被实现
// ShutdownOption provides a way to configure properties of the shutdown
// process. Currently, no options have been implemented.
type ShutdownOption interface {
	apply(*shutdowner)
}

type shutdowner struct {
	app *App
}

// 广播一个signal给到application所有的Done channel并开始停止
// Shutdown broadcasts a signal to all of the application's Done channels
// and begins the Stop process.
func (s *shutdowner) Shutdown(opts ...ShutdownOption) error {
	return s.app.broadcastSignal(syscall.SIGTERM)
}

func (app *App) shutdowner() Shutdowner {
	return &shutdowner{app: app}
}

// 广播signal
func (app *App) broadcastSignal(signal os.Signal) error {
	app.donesMu.RLock()
	defer app.donesMu.RUnlock()

	var unsent int
	for _, done := range app.dones {
		select {
		case done <- signal:
		default:
			// shutdown called when done channel has already received a
			// termination signal that has not been cleared
			unsent++
		}
	}

	if unsent != 0 {
		return fmt.Errorf("failed to send %v signal to %v out of %v channels",
			signal, unsent, len(app.dones),
		)
	}

	return nil
}
