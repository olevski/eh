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

package eh

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
)

func divide(x int, y int) (int, error) {
	if y == 0 {
		return 0, fmt.Errorf("divide by zero")
	}
	return x / y, nil
}

func doDivide(x int, y int) (res Result[int]) {
	defer EscapeHatch(&res)
	return NewResult(divide(x, y))
}

func doDivideMultiple(x int, y int) (res Result[int]) {
	defer EscapeHatch(&res)
	val := NewResult(divide(x, y)).Eh()
	return NewResult(divide(val+2, y))
}

func doDivideGoRoutine(x int, y int, res *Result[int], wg *sync.WaitGroup) {
	defer wg.Done()
	*res = NewResult(divide(x, y))
}

func doDividePtr(x int, y int) (res Result[int]) {
	return NewResult(divide(x, y))
}

func TestSimple(t *testing.T) {
	res := doDivide(4, 2)
	if res.Err != nil {
		t.FailNow()
	}
	if res.Ok != 2 {
		t.FailNow()
	}
	res = doDivide(1, 0)
	if res.Err == nil {
		t.FailNow()
	}
	if res.Ok != 0 {
		t.FailNow()
	}
}

func TestSimpleResultPointer(t *testing.T) {
	res := doDivide(4, 2)
	if res.Err != nil {
		t.FailNow()
	}
	if res.Ok != 2 {
		t.FailNow()
	}
	res = doDivide(1, 0)
	if res.Err == nil {
		t.FailNow()
	}
	if res.Ok != 0 {
		t.FailNow()
	}
}

func TestGoRoutines(t *testing.T) {
	res1 := Result[int]{}
	res2 := Result[int]{}
	res3 := Result[int]{}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go doDivideGoRoutine(4, 2, &res1, &wg)
	go doDivideGoRoutine(4, 0, &res2, &wg)
	go doDivideGoRoutine(5, 0, &res3, &wg)
	wg.Wait()
	if res1.Ok != 2 {
		t.Fatalf("Res1.Ok != 2, it equals %v", res1.Ok)
	}
	if res2.Err == nil {
		t.Fatalf("Res2.Err != nil, result: %+v", res2)
	}
	if res3.Err == nil {
		t.Fatalf("Res3.Err != nil, result: %+v", res3)
	}
}

func TestSimpleResultMultipleOk(t *testing.T) {
	res := doDivideMultiple(4, 2)
	if res.Err != nil {
		t.Fatalf("Error should be nil but is %+v", res)
	}
	if res.Ok != 2 {
		t.Fatalf("Ok has unexpected value %+v", res)
	}
}

func TestSimpleResultMultipleFail(t *testing.T) {
	res := doDivideMultiple(4, 0)
	if res.Err == nil {
		t.Fatalf("Error should not be nil but is %+v", res)
	}
	if res.Ok != 0 {
		t.Fatalf("Ok has unexpected value %+v", res)
	}
}

func example(aFile string) (res Result[[]byte]) {
	defer EscapeHatch(&res)
	buff := make([]byte, 5)
	file := NewResult(os.Open(aFile)).Eh()
	_ = NewResult(file.Read(buff)).Eh()
	return Result[[]byte]{Ok: buff}
}

func TestExample(t *testing.T) {
	res := example("README.md")
	if res.IsErr() {
		t.Fatalf("Err is not nil %+v", res)
	}
}

func TestExampleFail(t *testing.T) {
	res := example("non-existing-file")
	if res.IsOk() {
		t.Fatalf("Err should be nil %+v", res)
	}
}

func TestMustUnwrap(t *testing.T) {
	res := Result[int]{Ok: 1}
	ok := res.MustUnwrap()
	if ok != 1 {
		t.Fatal("Unwrap should return 1")
	}
}

func TestMustUnwrapPanic(t *testing.T) {
	res := Result[int]{Err: fmt.Errorf("error")}
	defer func() { recover() }()
	_ = res.MustUnwrap()
	t.Fatal("code should have panicked")
}

func TestMustUnwrapErr(t *testing.T) {
	aErr := fmt.Errorf("error")
	res := Result[int]{Err: aErr}
	err := res.MustUnwrapErr()
	if err != aErr {
		t.Fatal("UnwrapErr should return error")
	}
}

func TestMustUnwrapErrPanic(t *testing.T) {
	res := Result[int]{Ok: 1}
	defer func() { recover() }()
	_ = res.MustUnwrapErr()
	t.Fatal("code should have panicked")
}

func TestEscapeHatchErrOk(t *testing.T) {

	divideSuccess := func() (_ok int, _err error) {
		defer EscapeHatchErr(&_err)
		val := NewResult(divide(4, 2)).Eh()
		return val, nil
	}

	if val, err := divideSuccess(); val != 2 || err != nil {
		t.Fatalf("divide result should be %d and should not have err", 2)
	}
}

func TestEscapeHatchErrFail(t *testing.T) {

	divideFail := func() (_ok int, _err error) {
		defer EscapeHatchErr(&_err)
		val := NewResult(divide(4, 0)).Eh()
		return val, nil
	}

	if _, err := divideFail(); err.Error() != "divide by zero" {
		t.Fatal("an error of 'divide by zero' should be captured")
	}
}

func TestFallback(t *testing.T) {

	fallbackVal := 100

	divideFailWithFallback := func() (r Result[int]) {
		defer Fallback(&r, fallbackVal)
		val := NewResult(divide(4, 0)).Eh()
		return Result[int]{Ok: val}
	}

	result := divideFailWithFallback()

	if !result.IsOk() && result.MustUnwrap() != fallbackVal {
		t.FailNow()
	}

}

func TestFallbackAllError(t *testing.T) {

	errFailFirstTime := errors.New("fail first time")
	errFailSecondTime := errors.New("fail second time")
	errFailThirdTime := errors.New("fail third time")

	fallbackVal := 100

	funcMayFail := func() (int, error) {
		return 0, errFailFirstTime
	}

	divideFailWithErrorHandled := func() (r Result[int]) {
		defer Fallback(&r, fallbackVal, errFailThirdTime, errFailSecondTime, errFailFirstTime)

		val := NewResult(funcMayFail()).Eh()

		return Result[int]{Ok: val}
	}

	result := divideFailWithErrorHandled()

	if !result.IsOk() && result.Ok != fallbackVal {
		t.FailNow()
	}

}

func TestHandleMultipleError(t *testing.T) {

	errFailFirstTime := errors.New("fail first time")
	errFailSecondTime := errors.New("fail second time")
	errFailThirdTime := errors.New("fail third time")

	fallbackVal := 100

	funcMayFail := func() (int, error) {
		return 0, errFailFirstTime
	}

	divideFailMultipleError := func() (r Result[int]) {
		defer Fallback(&r, fallbackVal, errFailThirdTime)
		defer CatchError(&r, func(err error) int {
			t.Log(err.Error())
			FromFailable(errFailThirdTime).Eh()
			return 0
		}, errFailSecondTime)
		defer CatchError(&r, func(err error) int {
			t.Log(err.Error())
			FromFailable(errFailSecondTime).Eh()
			return 0
		}, errFailFirstTime)

		val := NewResult(funcMayFail()).Eh()

		return Result[int]{Ok: val}
	}

	result := divideFailMultipleError()

	if !result.IsOk() && result.Ok != fallbackVal {
		t.FailNow()
	}

}

func TestHandleAllError(t *testing.T) {

	errFailFirstTime := errors.New("fail first time")
	errFailSecondTime := errors.New("fail second time")
	errFailThirdTime := errors.New("fail third time")

	fallbackVal := 100

	funcMayFail := func() (int, error) {
		return 0, errFailFirstTime
	}

	divideFailWithErrorHandled := func() (r Result[int]) {
		defer CatchError(&r, func(err error) int {
			return fallbackVal
		}, errFailThirdTime, errFailSecondTime, errFailFirstTime)

		val := NewResult(funcMayFail()).Eh()

		return Result[int]{Ok: val}
	}

	result := divideFailWithErrorHandled()

	if !result.IsOk() && result.Ok != fallbackVal {
		t.FailNow()
	}

}

func TestHandleAnyError(t *testing.T) {

	fallbackVal := 100

	funcMayFail := func() (int, error) {
		return 0, fmt.Errorf("I am an arbitrary error")
	}

	divideFailWithErrorHandled := func() (r Result[int]) {
		defer CatchError(&r, func(err error) int {
			return fallbackVal
		})

		val := NewResult(funcMayFail()).Eh()

		return Result[int]{Ok: val}
	}

	result := divideFailWithErrorHandled()

	if !result.IsOk() && result.Ok != fallbackVal {
		t.FailNow()
	}

}
