package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	python3 "github.com/go-python/cpy3"
)

func main() {
	var err error

	//  At the end of the main function execution
	//  it will check, print and exit with non-zero code if any error is here
	defer func() {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	//  it will undo all initializations made by Py_Initialize()
	defer python3.Py_Finalize()
	//  it will print any python error if it is here
	//    no needs to call it after each single check of PyErr_Occurred()
	defer python3.PyErr_Print()

	//  Initialize the Python interpreter and create the GIL explicitly
	python3.Py_Initialize()

	if !python3.Py_IsInitialized() {
		err = errors.New("error initializing the python interpreter")
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	nameModule := "foo"
	fooModule := python3.PyImport_ImportModule(nameModule) //  New reference
	if fooModule == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error importing the python module")
		return
	}
	defer fooModule.DecRef()

	oddsAttr := "print_odds"
	evenAttr := "print_even"
	odds := fooModule.GetAttrString(oddsAttr) //  New reference
	if odds == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error getting the attribute print_odds")
		return
	}
	defer odds.DecRef()

	even := fooModule.GetAttrString(evenAttr) //  New reference
	if even == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error getting the attribute print_even")
		return
	}
	defer even.DecRef()

	limit := python3.PyLong_FromGoInt(50) //  New reference, will stolen later
	if limit == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error creating python long object")
		return
	}

	args := python3.PyTuple_New(1) //  New reference
	if args == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error creating python tuple object")
		return
	}
	defer args.DecRef()

	ret := python3.PyTuple_SetItem(args, 0, limit) //  Steals reference to limit
	limit = nil
	if ret != 0 {
		err = errors.New("error setting a tuple item")
		limit.DecRef()
		limit = nil
		return
	}

	//  Save the current state and release the lock
	//  so that goroutines can acquire it
	state := python3.PyEval_SaveThread()

	go func() {
		_gstate := python3.PyGILState_Ensure()
		odds.Call(args, python3.PyDict_New())
		python3.PyGILState_Release(_gstate)

		wg.Done()
	}()

	go func() {
		_gstate := python3.PyGILState_Ensure()
		even.Call(args, python3.PyDict_New())
		python3.PyGILState_Release(_gstate)

		wg.Done()
	}()

	wg.Wait()

	//  Restore the state and lock the GIL
	python3.PyEval_RestoreThread(state)
}
