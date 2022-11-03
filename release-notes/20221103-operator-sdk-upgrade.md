<!-- markdownlint-disable MD041 -->
* The project was rebuilt using Operator SDK version 1.23.0.
* The operator APIs have been converted to single-group; projects depending on these will need to replace
`github.com/submariner-io/submariner-operator/api/submariner/v1alpha1` with 
`github.com/submariner-io/submariner-operator/api/v1alpha1` in their imports.