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

package fxreflect

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"go.uber.org/dig"
)

// fxreflect主要用于完成针对function的：
//    1、获取function的输出参数的类型(包括其内嵌的成员字段的类型甚至子类型)
//    2、获取function的整个调用链，以及每个调用帧所对应的函数名(完整的路径：vender/xxx/xxx/xxx.method等【会注意反转义的处理】)

// Match from beginning of the line until the first `vendor/` (non-greedy)
var vendorRe = regexp.MustCompile("^.*?/vendor/")

// ReturnTypes takes a func and returns a slice of string'd types.
func ReturnTypes(t interface{}) []string {
	if reflect.TypeOf(t).Kind() != reflect.Func {
		// Invalid provide, will be logged as an error.
		return []string{}
	}

	rtypes := []string{}
	ft := reflect.ValueOf(t).Type()

	for i := 0; i < ft.NumOut(); i++ {  // 获取函数中所有的输出参数对应的类型(包括内嵌的字段成员的类型)
		t := ft.Out(i)

		traverseOuts(key{t: t}, func(s string) {  // 具体完成查找输出参数的类型及其内嵌类型
			rtypes = append(rtypes, s)
		})
	}

	return rtypes
}

// 函数输出参数
type key struct {
	t    reflect.Type
	name string
}

func (k *key) String() string {
	if k.name != "" {
		return fmt.Sprintf("%v:%s", k.t, k.name)
	}
	return k.t.String()
}


// 迭代找出函数中所有参数包含的所有的字段成员
func traverseOuts(k key, f func(s string)) {
	// skip errors
	if isErr(k.t) {
		return
	}

	// call funtion on non-Out types
	if dig.IsOut(k.t) {
		// keep recursing down on field members in case they are ins
		for i := 0; i < k.t.NumField(); i++ {  // 返回参数类型可能会存在成员嵌套类型 需要迭代遍历查找
			field := k.t.Field(i)
			ft := field.Type

			if field.PkgPath != "" { // 排除返回参数类型中的不可导出类型的成员
				continue // skip private fields
			}

			// keep recursing to traverse all the embedded objects
			k := key{
				t:    ft,
				name: field.Tag.Get("name"),
			}
			traverseOuts(k, f)
		}

		return
	}

	f(k.String())
}

// sanitize makes the function name suitable for logging display. It removes
// url-encoded elements from the `dot.git` package names and shortens the
// vendored paths.
// 提供合适的function在log中显示名称：vender/xxx/xxx/xxx.method（并对import包路径进行反转义处理）
func sanitize(function string) string {
	// Use the stdlib to un-escape any package import paths which can happen
	// in the case of the "dot-git" postfix. Seems like a bug in stdlib =/
	if unescaped, err := url.QueryUnescape(function); err == nil { // 解决导入路径的反转义类似.git结尾的import路径会存在问题
		function = unescaped
	}

	// strip everything prior to the vendor
	return vendorRe.ReplaceAllString(function, "vendor/") // 提取出来 vendor/在内后续的内容
}

// Caller returns the formatted calling func name
// 对调用函数名称进行格式化: 输出函数调用链(会剔除本框架内调用链)
func Caller() string {
	// Ascend at most 8 frames looking for a caller outside fx.
	pcs := make([]uintptr, 8)

	// Don't include this frame.
	n := runtime.Callers(2, pcs) // 剔除本框架的调用
	if n == 0 {
		return "n/a"
	}

	frames := runtime.CallersFrames(pcs)  // 获取到调用链
	for f, more := frames.Next(); more; f, more = frames.Next() {  // 获取不同的函数调用帧
		if shouldIgnoreFrame(f) {
			continue
		}
		return sanitize(f.Function)  // 函数完整路径
	}
	return "n/a"
}

// FuncName returns a funcs formatted name
// 格式化后的函数名
func FuncName(fn interface{}) string {
	fnV := reflect.ValueOf(fn)
	if fnV.Kind() != reflect.Func {  // 只针对function
		return "n/a"
	}

	function := runtime.FuncForPC(fnV.Pointer()).Name() // 根据指定的指针获取具体的调用帧函数
	return fmt.Sprintf("%s()", sanitize(function)) // 输出：类似vender/xxx/xxx/xxx.function()
}

// 是否实现error接口
func isErr(t reflect.Type) bool {
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	return t.Implements(errInterface)
}


// 追踪调用链直至离开fx框架；这样能避免通过硬编码跳过调用帧，同时也能在包装的情况下 对应的代码也能运行的很好
// Ascend the call stack until we leave the Fx production code. This allows us
// to avoid hard-coding a frame skip, which makes this code work well even
// when it's wrapped.
func shouldIgnoreFrame(f runtime.Frame) bool {
	if strings.Contains(f.File, "_test.go") {
		return false
	}
	if strings.Contains(f.File, "go.uber.org/fx") {
		return true
	}
	return false
}
