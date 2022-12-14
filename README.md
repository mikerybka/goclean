# goclean

`goclean` reads all the Go code in a package using `go/ast`.
It then replaces your hand-written `.go` files into many generated `.go` files named after each of your top-level declarations.
Each declaration in your package, whether exported or not, is given its own file named after itself.
The only exception is methods.
Methods are placed in the same file as their type declaration.
