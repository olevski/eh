// Copyright © 2023 Tasko Olevski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package eh (escapehatch) provides Rust-like error handling in Go.
package eh

import (
	"errors"
)

// Result represents a struct that contains an error in the Err field
// or the valid output from some action in the Ok field. Since Results are copied it is
// reccomended that the Ok value be a pointer.
type Result[T any] struct {
	Ok  T
	Err error
}

// NewResult creates a Result from any value and an error. You can use this to
// convert the output of any function that returns a value and an error.
func NewResult[T any](val T, err error) Result[T] {
	return Result[T]{val, err}
}

// FromFailable creates a Result from an error. This is useful when you need to
// execute a function that doesn't return a value but may result in failure.
//
// Example:
//
//	func example() (r eh.Result[string]) {
//		eh.EscapeHatch(&r)
//
//		var data string
//		eh.FromFailable(json.Unmarshal([]byte{}, &data)).Eh()
//
//		return eh.Result[string]{Ok: data}
//	}
func FromFailable(err error) Result[any] {
	return Result[any]{0, err}
}

// Eh checks if there is an error in the result and if so then it will
// panic with the error that was encountered. If there is no error the Ok value is returned.
func (r Result[T]) Eh() T {
	if r.Err != nil {
		panic(ehError{r.Err})
	}
	return r.Ok
}

// IsOk returns true when result has no error and otherwise false
func (r Result[T]) IsOk() bool {
	return r.Err == nil
}

// IsErr returns true when result has error and otherwise false
func (r Result[T]) IsErr() bool {
	return r.Err != nil
}

// MustUnwrap returns the Ok value or panics if there is an error.
func (r Result[T]) MustUnwrap() T {
	if r.Err != nil {
		panic(r.Err)
	}
	return r.Ok
}

// MustUnwrapErr returns the Err value or panics if there is no error.
func (r Result[T]) MustUnwrapErr() error {
	if r.Err == nil {
		panic("expected the result to contain error")
	}
	return r.Err
}

// Unwrap method returns a value and an error
func (r Result[T]) Unwrap() (T, error) {
	return r.Ok, r.Err
}

// Unwrap function unwraps a Result into a value and an error, making
// it useful for implementing inline callbacks.
//
// Example:
//
//	// from this:
//	newIterable := someIterable.Map(func (v Result[T]) {
//		 return v.Unwrap()
//	 })
//	// to this:
//	newIterable := someIterable.Map(eh.Unwrap[T])
func Unwrap[T any](r Result[T]) (T, error) {
	return r.Ok, r.Err
}

// ehError is used to wrap any errors that are raised because of calling
// ReturnIfErr on a Result.
type ehError struct {
	error
}

// EscapeHatch will recover from a panic that was raised from any error
// raised from the error checks performed by eh. The recovered error is
// populated in the Result pointed by the res pointer. If the recovered
// error was not raised by eh then the same panic will be raised.
func EscapeHatch[T any](res *Result[T]) {
	if r := recover(); r != nil {
		err, ok := r.(ehError)
		if !ok {
			// Panicking again because the recovered panic is not an ehError
			panic(r)
		}
		*res = Result[T]{Err: err.error}
	}
}

// EscapeHatchErr is similarly to the `EscapeHatch`, with the difference
// that it is designed for implementing methods that must conform to return
// signatures of `(value, error)` or `(error)`, such as overriding a method
// in a superclass.
//
// Example:
//
//	func ExampleRequestHandler(c *Context) (val string, err error) {
//		defer eh.EscapeHatchErr(&err)	// adapt to (val, error)
//
//		successVal := eh.NewResult(failableFunc()).Eh()
//
//		return successVal, nil
//	}
func EscapeHatchErr(err *error) {
	res := FromFailable(*err)
	defer func() {
		_, *err = res.Unwrap()
	}()
	defer EscapeHatch(&res)
	if r := recover(); r != nil {
		panic(r)
	}
}

// HandlerError is similar to `Fallback`, with the difference that it
// executes a handler with the fallback value returned instead of directly
// specifies the value. Furthermore, you can call `.Eh()` within the
// handler, which may succeed or rethrow a new error to later handlers.
//
// Example:
//
//	func main() {
//		var WhenGetFromDBError error
//		var WhenGetFromRemoteError error
//		var WhenGetFromInMemoryError error
//		func Example() (r eh.Result[string]) {
//				// Escape if error is not handled
//				defer eh.EscapeHatch(&r)
//				// Run error handler when previous `defer` rethrows error
//				defer eh.CatchError(&r, func(_ error) {
//					return eh.NewResult(GetFromRemote()).Eh()
//				}, WhenGetFromDBError)
//				// Run error handler when error is `WhenGetFromInMemoryError`
//				defer eh.CatchError(&r, func(_ error) {
//					return eh.NewResult(GetFromDb()).Eh()
//				}, WhenGetFromInMemoryError)
//				successVal := eh.NewResult(GetFromInMemory()).Eh()
//				return eh.Result[string]{Ok: successVal}
//			}
//	}
func CatchError[T any](res *Result[T], catcher func(error) T, when ...error) {
	defer func() {
		if res.IsOk() {
			return
		}

		err := res.MustUnwrapErr()
		// Passing nil `when` means use orElse for any error
		if when == nil {
			*res = Result[T]{Ok: catcher(err)}
			return
		}
		for _, target := range when {
			if !errors.Is(err, target) {
				continue
			}
			// Use fallback value if the result error matches the target error, otherwise leave the result untouched
			*res = Result[T]{Ok: catcher(err)}
			break
		}
	}()
	defer EscapeHatch(res)
	if r := recover(); r != nil {
		panic(r)
	}
}

// Fallback allows for the substitution of an error with a default value.
// It is optional to specify the types of errors for which the default value
// will be used. If no errors are specified, the default value will be used
// as a fallback for all errors.
//
// Example:
//
//	func main() {
//		var WhenGetFromDBError error
//		var WhenGetFromRemoteError error
//		var WhenGetFromInMemoryError error
//		func Example() (r eh.Result[string]) {
//				// Escape when error is not handled
//				defer eh.EscapeHatch(&r)
//				// Fallback to default value
//				defer eh.Fallback(&r, "default value",
//					WhenGetFromDBError,
//					WhenGetFromRemoteError,
//					WhenGetFromInMemoryError)
//
//				successVal := eh.NewResult(FailableApiToGetData()).Eh()
//				return eh.Result[string]{Ok: successVal}
//			}
//	}
func Fallback[T any](res *Result[T], fallback T, when ...error) {
	defer CatchError(res, func(_ error) T { return fallback }, when...)
	if r := recover(); r != nil {
		panic(r)
	}
}
