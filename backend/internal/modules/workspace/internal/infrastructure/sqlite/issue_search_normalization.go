package sqlite

import (
	"database/sql/driver"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	moderncsqlite "modernc.org/sqlite"
)

func init() {
	moderncsqlite.MustRegisterDeterministicScalarFunction("goclaw_issue_search_normalize", 1, func(_ *moderncsqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		if len(args) != 1 || args[0] == nil {
			return "", nil
		}
		value, ok := args[0].(string)
		if !ok {
			return "", nil
		}
		return application.NormalizeIssueSearchText(value), nil
	})
}
