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
	if res.Err != nil {
		t.Fatalf("Err is not nil %+v", res)
	}
}

func TestExampleFail(t *testing.T) {
	res := example("non-existing-file")
	if res.Err == nil {
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
