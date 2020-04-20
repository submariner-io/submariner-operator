package controller

import (
	"github.com/submariner-io/submariner-operator/pkg/controller/servicediscovery"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, servicediscovery.Add)
}
