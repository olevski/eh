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

// ReturnIfErr checks if there is an error in the result and if so then it will
// panic with the error that was encountered.
func (r Result[T]) ReturnIfErr() Result[T] {
	if r.Err != nil {
		panic(Error{r.Err})
	}
	return r
}

// Error is used to wrap any errors that are raised because of calling
// ReturnIfErr on a Result.
type Error struct {
	WrappedErr error
}

func (e Error) Error() string {
	return e.Error()
}

// EscapeHatch will recover from a panic that was raised from any error
// raised from the error checks performed by eh. The recovered error is
// populated in the Result pointed by the res pointer. If the recovered
// error was not raised by eh then the same panic will be raised.
func EscapeHatch[T any](res *Result[T]) {
	if r := recover(); r != nil {
		err, ok := r.(Error)
		if !ok {
			// Panicking again because the recovered panic is not an Error
			panic(r)
		}
		*res = Result[T]{Err: err.WrappedErr}
	}
}
