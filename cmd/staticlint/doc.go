// staticlint is a linter for Go source code.
//
// Представляет из себя multichecker, состоящий из:
//   - стандартных статических анализаторов пакета golang.org/x/tools/go/analysis/passes;
//   - всех анализаторов класса SA пакета staticcheck.io;
//   - один анализатор ST1015;
//   - анализатор ErrCheck github.com/Antonboom/errname;
//   - анализатор bodyclose github.com/timakin/bodyclose/passes/bodyclose;
//
// запуск: staticlint ./...
package main
