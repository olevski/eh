# eh (Escape Hatch)

Succinct error handling for Go inspired by Rust. `eh` can help you avoid the usual Go error 
handling boileplate.

This library tries to provide similar functionality to Rust's Result enum and
the `?` operator.

For example turn this:

```go
func example(aFile string) []byte, error {
    f, err := os.Open(aFile)
    if err != nil {
        return nil, err
    }
    buff := make([]byte, 5)
    _, err := f.Read(b1)
    if err != nil {
        return nil, err
    }
    return buff, nil
}
```

Into this:

```go
func example(aFile string) (res eh.Result[[]byte]) {
	defer eh.EscapeHatch(&res)  // must have pointer to the named output Result
	buff := make([]byte, 5)
	file := eh.NewResult(os.Open(aFile)).Eh()
	_ = eh.NewResult(file.Read(buff)).Eh()
	return eh.Result[[]byte]{Ok: buff}
}
```

When an error occurs (for example if the file does not exist) this is what happens:

```
{Ok:[] Err:open non-existent-file.md: no such file or directory}
```

For a successful operation the output is like this:

```
{Ok:[60 33 100 111 99 116 121 112] Err: <nil>}
```

The library provides just a handful or public structs and functions:
- `Result` structs to capture an outcome that can potentially result in an error
- `Eh` method (`?` was not possible) that will stop the current function from executing
  if there is an error and return a `Result` that contains the error. If there is no error the `Result`
  is unwrapped and the `Ok` value is returned. Just like Canadians add "eh" to the end of a sentence 
  to make it a question, you can call `Eh` on a `Result` to get a similar functionality to the question
  mark operator.
- `EscapeHatch` function that should be deferred and given access to the enclosing
  function named `Result` argument so that the error can be recovered and the 
  enclosed function can be interrupted early if an error occurs.

## Instructions

A few simple steps to incorporate in your code:
1. Return a named `Result` from your fuction
2. `defer` the `EscapeHatch` function with pointer to the named `Result` argument
3. Wrap functions that return `any, error` into `eh.NewResult` to get `Result`
4. Call `Eh()` on any `Result` to stop the execution if there is an error 
  and return `Result{Err: Error}` from the enclosing function.
