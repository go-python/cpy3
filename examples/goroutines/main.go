package main

import (
	"fmt"
	"os"
	"sync"

	python3 "github.com/go-python/cpy3"
)

func main() {
	// The following will also create the GIL explicitly
	// by calling PyEval_InitThreads(), without waiting
	// for the interpreter to do that
	python3.Py_Initialize()

	if !python3.Py_IsInitialized() {
		fmt.Println("Error initializing the python interpreter")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	nameModule := "foo"
	fooModule := python3.PyImport_ImportModule(nameModule)
	if fooModule == nil {
		fmt.Printf("The module %s doesn't exist\n", nameModule)
	}
	oddsAttr := "print_odds"
	evenAttr := "print_even"

	if !fooModule.HasAttrString(oddsAttr) {
		fmt.Printf("The module doesn't have attribute %s\n", oddsAttr)
		os.Exit(1)
	}

	if !fooModule.HasAttrString(evenAttr) {
		fmt.Printf("The module doesn't have attribute %s\n", evenAttr)
		os.Exit(1)
	}

	odds := fooModule.GetAttrString(oddsAttr)
	even := fooModule.GetAttrString(evenAttr)

	if odds == nil || even == nil {
		panic("Error importing function")
	}

	// Py_Initialize() has locked the the GIL but at this point we don't need it
	// anymore. We save the current state and release the lock
	// so that goroutines can acquire it
	state := python3.PyEval_SaveThread()

	go func() {
		_gstate := python3.PyGILState_Ensure()
		odds.Call(python3.PyTuple_New(0), python3.PyDict_New())
		python3.PyGILState_Release(_gstate)

		wg.Done()
	}()

	go func() {
		_gstate := python3.PyGILState_Ensure()
		even.Call(python3.PyTuple_New(0), python3.PyDict_New())
		python3.PyGILState_Release(_gstate)

		wg.Done()
	}()

	wg.Wait()

	// At this point we know we won't need python anymore in this
	// program, we can restore the state and lock the GIL to perform
	// the final operations before exiting.
	python3.PyEval_RestoreThread(state)
	python3.Py_Finalize()
}
