package main

import (
	"errors"
	"log"
	"os"
	"runtime"
	"sync"

	python3 "github.com/go-python/cpy3"
)

func main() {
	var err error

	// At the end of all, if there was an error
	// prints the error and exit with the non-zero code
	defer func() {
		if err != nil {
			log.Printf("%+v", err)
			os.Exit(1)
		}
	}()

	// Undo all initializations made by Py_Initialize() and subsequent
	defer python3.Py_Finalize()

	// Prints any python error if it was here
	// no needs to call it after each single check of PyErr_Occurred()
	defer python3.PyErr_Print()

	// Initialize the Python interpreter and
	// since version 3.7 it also create the GIL explicitly by calling PyEval_InitThreads()
	// so you don’t have to call PyEval_InitThreads() yourself anymore
	python3.Py_Initialize() // create the GIL, the GIL is locked by the main thread

	if !python3.Py_IsInitialized() {
		err = errors.New("error initializing the python interpreter")
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	fooModule := python3.PyImport_ImportModule("foo") // new reference, a call DecRef() is needed
	if fooModule == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error importing the python module")
		return
	}
	defer fooModule.DecRef()

	odds := fooModule.GetAttrString("print_odds") // new reference, a call DecRef() is needed
	if odds == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error getting the attribute print_odds")
		return
	}
	defer odds.DecRef()

	even := fooModule.GetAttrString("print_even") // new reference, a call DecRef() is needed
	if even == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error getting the attribute print_even")
		return
	}
	defer even.DecRef()

	limit := python3.PyLong_FromGoInt(50) // new reference, will stolen later, a call DecRef() is NOT needed
	if limit == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error creating python long object")
		return
	}

	args := python3.PyTuple_New(1) // new reference, a call DecRef() is needed
	if args == nil && python3.PyErr_Occurred() != nil {
		err = errors.New("error creating python tuple object")
		return
	}
	defer args.DecRef()

	ret := python3.PyTuple_SetItem(args, 0, limit) // steals reference to limit

	// Cleans the Go variable, because now a new owner is caring about related PyObject
	// no action, such as a call DecRef(), is needed here
	limit = nil

	if ret != 0 {
		err = errors.New("error setting a tuple item")
		limit.DecRef()
		limit = nil
		return
	}

	// Save the current state and release the GIL
	// so that goroutines can acquire it
	state := python3.PyEval_SaveThread() // release the GIL, the GIL is unlocked for using by goroutines

	go func() {
		runtime.LockOSThread()
		_gstate := python3.PyGILState_Ensure() // acquire the GIL, the GIL is locked by the 1st goroutine
		odds.Call(args, python3.PyDict_New())
		python3.PyGILState_Release(_gstate) // release the GIL, the GIL is unlocked for using by others

		wg.Done()
	}()

	go func() {
		runtime.LockOSThread()
		_gstate := python3.PyGILState_Ensure() // acquire the GIL, the GIL is locked by the 2nd goroutine
		even.Call(args, python3.PyDict_New())
		python3.PyGILState_Release(_gstate) // release the GIL, the GIL is unlocked for using by others

		wg.Done()
	}()

	wg.Wait()

	// Restore the state and lock the GIL
	python3.PyEval_RestoreThread(state) // acquire the GIL, the GIL is locked by the main thread
}
