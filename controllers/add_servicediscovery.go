package controllers

import (
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, servicediscovery.Add)
}
