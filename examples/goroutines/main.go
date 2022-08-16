package main

import (
	"errors"
	"log"
	"runtime"
	"sync"

	python3 "github.com/go-python/cpy3"
)

var wg sync.WaitGroup

func isPyErr(val *python3.PyObject) bool {

	if val == nil && python3.PyErr_Occurred() != nil {
		python3.PyErr_Print()
		return true
	}

	return false
}

func callPyFunc(pyFunc *python3.PyObject, args *python3.PyObject, kwargs *python3.PyObject) {
	runtime.LockOSThread()

	_gstate := python3.PyGILState_Ensure()    // acquire the GIL, the GIL is locked by the 1st goroutine
	defer python3.PyGILState_Release(_gstate) // release the GIL, the GIL is unlocked for using by others

	ret := pyFunc.Call(args, kwargs)
	if isPyErr(ret) {
		return
	}
	defer ret.DecRef()

	wg.Done()
}

func main() {
	// Undo all initializations made by Py_Initialize() and subsequent
	defer python3.Py_Finalize()

	// Prints any python error if it was here
	// no needs to call it after each single check of PyErr_Occurred()
	// defer python3.PyErr_Print()

	// Initialize the Python interpreter and
	// since version 3.7 it also create the GIL explicitly by calling PyEval_InitThreads()
	// so you don’t have to call PyEval_InitThreads() yourself anymore
	python3.Py_Initialize() // create the GIL, the GIL is locked by the main thread

	if !python3.Py_IsInitialized() {
		log.Printf("%+v", errors.New("error initializing the python interpreter"))
		return
	}

	fooModule := python3.PyImport_ImportModule("foo") // new reference, a call DecRef() is needed
	if isPyErr(fooModule) {
		return
	}
	defer fooModule.DecRef()

	odds := fooModule.GetAttrString("print_odds") // new reference, a call DecRef() is needed
	if isPyErr(odds) {
		return
	}
	defer odds.DecRef()

	even := fooModule.GetAttrString("print_even") // new reference, a call DecRef() is needed
	if isPyErr(even) {
		return
	}
	defer even.DecRef()

	limit := python3.PyLong_FromGoInt(50) // new reference, will stolen later, a call DecRef() is NOT needed
	if limit == nil {
		log.Printf("%+v", errors.New("error creating python long object"))
		return
	}

	args := python3.PyTuple_New(1) // new reference, a call DecRef() is needed
	if args == nil {
		log.Printf("%+v", errors.New("error creating python tuple object"))
		return
	}
	defer args.DecRef()

	ret := python3.PyTuple_SetItem(args, 0, limit) // steals reference to limit
	if ret != 0 {
		log.Printf("%+v", errors.New("error setting a tuple item"))
		limit.DecRef()
		limit = nil
		return
	}
	// Cleans the Go variable, because now a new owner is caring about related PyObject
	// no action, such as a call DecRef(), is needed here
	limit = nil

	kwargs := python3.PyDict_New()
	if kwargs == nil {
		log.Printf("%+v", errors.New("error initializing kwargs in callPyFunc "))
		return
	}
	defer kwargs.DecRef()

	// Save the current state and release the GIL
	// so that goroutines can acquire it
	state := python3.PyEval_SaveThread() // release the GIL, the GIL is unlocked for using by goroutines

	wg.Add(2)

	go callPyFunc(odds, args, kwargs)

	go callPyFunc(even, args, kwargs)

	wg.Wait()

	// Restore the state and lock the GIL
	python3.PyEval_RestoreThread(state) // acquire the GIL, the GIL is locked by the main thread
}
