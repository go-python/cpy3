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

	// At the end of the main function execution
	// it will check, print and exit with non-zero code if any error is here
	defer func() {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	// it will undo all initializations made by Py_Initialize()
	defer python3.Py_Finalize()
	// it will print any python error if it is here
	//   no needs to call it after each single check of PyErr_Occurred()
	defer python3.PyErr_Print()

	// The following will also create the GIL explicitly
	// by calling PyEval_InitThreads(), without waiting
	// for the interpreter to do that
	python3.Py_Initialize()

	if !python3.Py_IsInitialized() {
		err = errors.New("Error initializing the python interpreter")
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	nameModule := "foo"
	fooModule := python3.PyImport_ImportModule(nameModule)
	if fooModule == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("Error importing the python module")
		fooModule.DecRef()
		return
	}

	oddsAttr := "print_odds"
	evenAttr := "print_even"

	odds := fooModule.GetAttrString(oddsAttr)
	if odds == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("Error getting the attribute print_odds")
		odds.DecRef()
		return
	}

	even := fooModule.GetAttrString(evenAttr)
	if even == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("Error getting the attribute print_even")
		even.DecRef()
		return
	}

	limit := python3.PyLong_FromGoInt(50)
	if limit == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("Error creating python long object")
		limit.DecRef()
		return
	}

	args := python3.PyTuple_New(1)
	if args == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("Error creating python tuple object")
		args.DecRef()
		return
	}

	ret := python3.PyTuple_SetItem(args, 0, limit)
	if ret != 0 {
		args.DecRef()
		limit.DecRef()
		err = errors.New("Error setting a tuple item")
		return
	}

	// Py_Initialize() has locked the the GIL but at this point we don't need it
	// anymore. We save the current state and release the lock
	// so that goroutines can acquire it
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

	// At this point we know we won't need python anymore in this
	// program, we can restore the state and lock the GIL to perform
	// the final operations before exiting.
	python3.PyEval_RestoreThread(state)
}
