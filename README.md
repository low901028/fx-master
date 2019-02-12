# :unicorn: Fx [![GoDoc][doc-img]][doc] [![Github release][release-img]][release] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Go Report Card][report-card-img]][report-card]

go依赖注入框架：
An application framework for Go that:

* Makes dependency injection easy. 依赖注入使用变得容易
* Eliminates the need for global state and `func init()`. 不需要使用全局state和func init()来完成相关的操作

## Installation 安装说明

We recommend locking to [SemVer](http://semver.org/) range `^1` using [Glide](https://github.com/Masterminds/glide):

```
glide get 'go.uber.org/fx#^1'
```

## Stability 稳定性

This library is `v1` and follows [SemVer](http://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0`.

This project follows the [Go Release Policy][release-policy]. Each major
version of Go is supported until there are two newer major releases.

[doc-img]: http://img.shields.io/badge/GoDoc-Reference-blue.svg
[doc]: https://godoc.org/go.uber.org/fx

[release-img]: https://img.shields.io/github/release/uber-go/fx.svg
[release]: https://github.com/uber-go/fx/releases

[ci-img]: https://img.shields.io/travis/uber-go/fx/master.svg
[ci]: https://travis-ci.org/uber-go/fx/branches

[cov-img]: https://codecov.io/gh/uber-go/fx/branch/dev/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/fx/branch/dev

[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/fx
[report-card]: https://goreportcard.com/report/github.com/uber-go/fx

[release-policy]: https://golang.org/doc/devel/release.html#policy
