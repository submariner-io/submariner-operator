package controller

import (
	"github.com/submariner-io/submariner-operator/pkg/controller/serviceimport"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, serviceimport.Add)
}
