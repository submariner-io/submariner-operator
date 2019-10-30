package main

import "github.com/submariner-io/submariner-operator/pkg/subctl/cmd"

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err.Error())
	}
}
