# Field Name Checker

Some functions in the Terraform SDK that take as arguments the names of configuration fields will silently fail if
given input that does not match the name of any fields in the package. This program detects if the string
arguments to these functions do not match any field names in the same packages.

A zero exit code indicates no problems found. Any other code, along with output, indicates that a discrepancy was found.

These functions include:

* HasChange()
* HasChanges()
* GetChange()
* Set()
* Get()
* GetOk()

## Example

```
$ field-name-checker ./internal/services/batch
./internal/services/batch/batch_pool_data_source.go:778:11: schema field/component 'mount_configuration' not found in package
```

It must be executed from the root of the program so it can access the `go.mod` file belonging to the program and properly build the abstract syntax tree. Note, it only performs crude checks on nested properties, i.e. ensures that every nested property is found somewhere in the same package. It does NOT check for proper nesting relationships.
